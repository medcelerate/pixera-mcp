// Package pixera is a client for the Pixera Native API
// (https://help.pixera.one/api). The API is JSON-RPC 2.0 spoken over a TCP
// socket on which Pixera acts as the server (default port 1400). In the
// "TCP(dl)" framing used here, each JSON message is terminated by the literal
// 4-byte ASCII delimiter "0xPX".
//
// The client's target (host/port) is repointable at runtime so the bridge can
// be aimed at a different Pixera server without a restart.
package pixera

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// delimiter terminates every message in Pixera's TCP(dl) framing.
var delimiter = []byte("0xPX")

// Target identifies a Pixera server's API endpoint.
type Target struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

func (t Target) withDefaults() Target {
	if t.Port == 0 {
		t.Port = 1400
	}
	return t
}

// Address renders host:port.
func (t Target) Address() string {
	t = t.withDefaults()
	return net.JoinHostPort(t.Host, fmt.Sprintf("%d", t.Port))
}

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("pixera error %d: %s", e.Code, e.Message)
}

// Client talks to a Pixera Native API over TCP. It is safe for concurrent use;
// its target can be changed while requests are in flight, and the connection is
// (re)established lazily.
type Client struct {
	target      atomic.Pointer[Target]
	dialTimeout time.Duration
	callTimeout time.Duration

	mu      sync.Mutex // guards conn setup/teardown
	conn    net.Conn
	writeMu sync.Mutex

	seq     atomic.Int64
	pending sync.Map // id -> chan *rpcResponse
}

// NewClient creates a client aimed at target.
func NewClient(target Target, callTimeout time.Duration) *Client {
	if callTimeout <= 0 {
		callTimeout = 15 * time.Second
	}
	c := &Client{dialTimeout: 10 * time.Second, callTimeout: callTimeout}
	t := target.withDefaults()
	c.target.Store(&t)
	return c
}

// Target returns the current target.
func (c *Client) Target() Target { return *c.target.Load() }

// SetTarget repoints the client and drops any existing connection so the next
// call redials.
func (c *Client) SetTarget(t Target) {
	t = t.withDefaults()
	c.target.Store(&t)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropConnLocked()
}

func (c *Client) dropConnLocked() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// ensureConn returns a live connection, dialing if needed.
func (c *Client) ensureConn(ctx context.Context) (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn, nil
	}
	target := c.Target()
	if target.Host == "" {
		return nil, fmt.Errorf("no pixera target configured")
	}
	d := net.Dialer{Timeout: c.dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", target.Address())
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target.Address(), err)
	}
	c.conn = conn
	go c.readLoop(conn)
	return conn, nil
}

// readLoop scans delimited messages from conn and dispatches responses by id.
func (c *Client) readLoop(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 16<<20)
	scanner.Split(splitOnDelimiter)
	for scanner.Scan() {
		msg := bytes.TrimSpace(scanner.Bytes())
		if len(msg) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue // ignore non-JSON / notifications we can't parse
		}
		if ch, ok := c.pending.LoadAndDelete(resp.ID); ok {
			r := resp
			ch.(chan *rpcResponse) <- &r
		}
	}
	// Connection closed: fail any in-flight callers and clear the conn.
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.mu.Unlock()
	c.pending.Range(func(key, value any) bool {
		if ch, ok := c.pending.LoadAndDelete(key); ok {
			close(ch.(chan *rpcResponse))
		}
		return true
	})
}

// splitOnDelimiter is a bufio.SplitFunc that yields the bytes before each 0xPX.
func splitOnDelimiter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.Index(data, delimiter); i >= 0 {
		return i + len(delimiter), data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Call invokes a JSON-RPC method and returns its raw result. params may be nil.
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	conn, err := c.ensureConn(ctx)
	if err != nil {
		return nil, err
	}

	id := c.seq.Add(1)
	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	payload = append(payload, delimiter...)

	ch := make(chan *rpcResponse, 1)
	c.pending.Store(id, ch)
	defer c.pending.Delete(id)

	c.writeMu.Lock()
	_, werr := conn.Write(payload)
	c.writeMu.Unlock()
	if werr != nil {
		c.mu.Lock()
		c.dropConnLocked()
		c.mu.Unlock()
		return nil, fmt.Errorf("write %s: %w", method, werr)
	}

	callCtx := ctx
	if c.callTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, c.callTimeout)
		defer cancel()
	}

	select {
	case <-callCtx.Done():
		return nil, fmt.Errorf("%s: %w", method, callCtx.Err())
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("%s: connection closed before reply", method)
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// Ping checks reachability by requesting the API revision.
func (c *Client) Ping(ctx context.Context) (json.RawMessage, error) {
	return c.Call(ctx, "Pixera.Utility.getApiRevision", map[string]any{})
}

// Close drops the connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropConnLocked()
}
