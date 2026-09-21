package pixera

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"
)

// fakePixera is a minimal TCP JSON-RPC server using the 0xPX framing. The
// handler receives the parsed request and returns (result, errorObj).
func fakePixera(t *testing.T, handle func(req map[string]any) (any, *rpcError)) (Target, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				sc := bufio.NewScanner(c)
				sc.Split(splitOnDelimiter)
				for sc.Scan() {
					var req map[string]any
					if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
						continue
					}
					result, rerr := handle(req)
					resp := map[string]any{"jsonrpc": "2.0", "id": req["id"]}
					if rerr != nil {
						resp["error"] = map[string]any{"code": rerr.Code, "message": rerr.Message}
					} else {
						resp["result"] = result
					}
					out, _ := json.Marshal(resp)
					out = append(out, delimiter...)
					_, _ = c.Write(out)
				}
			}(conn)
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return Target{Host: "127.0.0.1", Port: addr.Port}, func() { _ = ln.Close() }
}

func TestCallRoundTrip(t *testing.T) {
	var gotMethod string
	var gotParams map[string]any
	target, closeSrv := fakePixera(t, func(req map[string]any) (any, *rpcError) {
		gotMethod, _ = req["method"].(string)
		gotParams, _ = req["params"].(map[string]any)
		return 204, nil
	})
	defer closeSrv()

	c := NewClient(target, 5*time.Second)
	defer c.Close()

	raw, err := c.Call(context.Background(), "Pixera.Compound.setTransportModeOnTimeline",
		map[string]any{"timelineName": "Main", "mode": 1})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotMethod != "Pixera.Compound.setTransportModeOnTimeline" {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotParams["timelineName"] != "Main" || gotParams["mode"].(float64) != 1 {
		t.Fatalf("params = %v", gotParams)
	}
	if n, _ := strconv.Atoi(string(raw)); n != 204 {
		t.Fatalf("result = %s", raw)
	}
}

func TestCallReturnsRPCError(t *testing.T) {
	target, closeSrv := fakePixera(t, func(map[string]any) (any, *rpcError) {
		return nil, &rpcError{Code: -32601, Message: "method not found"}
	})
	defer closeSrv()
	c := NewClient(target, 5*time.Second)
	defer c.Close()
	if _, err := c.Call(context.Background(), "Bad.Method", nil); err == nil {
		t.Fatal("expected rpc error")
	}
}

func TestConcurrentCallsCorrelate(t *testing.T) {
	// Echo the requested "n" param back as the result, so mismatched
	// correlation would surface.
	target, closeSrv := fakePixera(t, func(req map[string]any) (any, *rpcError) {
		p, _ := req["params"].(map[string]any)
		return p["n"], nil
	})
	defer closeSrv()
	c := NewClient(target, 5*time.Second)
	defer c.Close()

	type res struct {
		n   int
		got float64
	}
	ch := make(chan res, 20)
	for i := 0; i < 20; i++ {
		go func(n int) {
			raw, err := c.Call(context.Background(), "Echo", map[string]any{"n": n})
			if err != nil {
				ch <- res{n, -1}
				return
			}
			var v float64
			_ = json.Unmarshal(raw, &v)
			ch <- res{n, v}
		}(i)
	}
	for i := 0; i < 20; i++ {
		r := <-ch
		if int(r.got) != r.n {
			t.Fatalf("correlation mismatch: sent %d got %v", r.n, r.got)
		}
	}
}

func TestSetTargetRepoints(t *testing.T) {
	c := NewClient(Target{Host: "10.0.0.1"}, time.Second)
	if got := c.Target().Address(); got != "10.0.0.1:1400" {
		t.Fatalf("default addr = %q", got)
	}
	c.SetTarget(Target{Host: "10.0.0.2", Port: 1412})
	if got := c.Target().Address(); got != "10.0.0.2:1412" {
		t.Fatalf("repointed addr = %q", got)
	}
}
