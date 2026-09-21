package pixera

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// hbHeader is the 8-byte packet header Pixera prefixes onto heartbeat datagrams:
// 4 ASCII bytes "pxr1" + a little-endian uint32 payload size.
var hbHeader = []byte("pxr1")

// LiveSystem describes one Pixera system reported in a heartbeat.
type LiveSystem struct {
	IP      string `json:"ip"`
	Name    string `json:"name"`
	IsLocal bool   `json:"isLocal"`
	State   string `json:"state"`
}

// Heartbeat is the parsed content of a Pixera heartbeat datagram.
type Heartbeat struct {
	ServerIP    string          `json:"serverIp"`
	APIPort     int             `json:"apiPort"`  // the JSON/TCP Native API port
	HTTPPort    int             `json:"httpPort"` // the HTTP/TCP port, if advertised
	LiveSystems []LiveSystem    `json:"liveSystems,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

// Target returns the Native API target the heartbeat points at.
func (h Heartbeat) Target() Target { return Target{Host: h.ServerIP, Port: h.APIPort} }

// hbEntry is one element of the heartbeat JSON array.
type hbEntry struct {
	Type       string       `json:"type"`
	IP         string       `json:"ip"`
	Port       int          `json:"port"`
	Protocol   string       `json:"protocol"`
	Heartbeats []LiveSystem `json:"heartbeats"`
}

// parseHeartbeat strips the optional pxr1 header and parses the JSON array.
func parseHeartbeat(data []byte) (*Heartbeat, error) {
	payload := data
	if len(data) >= 8 && string(data[:4]) == string(hbHeader) {
		size := int(binary.LittleEndian.Uint32(data[4:8]))
		payload = data[8:]
		if size > 0 && size <= len(payload) {
			payload = payload[:size]
		}
	}
	var entries []hbEntry
	if err := json.Unmarshal(payload, &entries); err != nil {
		return nil, fmt.Errorf("parse heartbeat json: %w", err)
	}
	hb := &Heartbeat{Raw: append(json.RawMessage(nil), payload...)}
	for _, e := range entries {
		switch e.Type {
		case "self":
			if e.IP != "" {
				hb.ServerIP = e.IP
			}
		case "protocol":
			switch e.Protocol {
			case "JSON/TCP":
				hb.APIPort = e.Port
			case "HTTP/TCP":
				hb.HTTPPort = e.Port
			}
		case "liveSystemHeartbeats":
			hb.LiveSystems = e.Heartbeats
		}
	}
	if hb.APIPort == 0 {
		return nil, fmt.Errorf("heartbeat has no JSON/TCP API port")
	}
	return hb, nil
}

// listenHeartbeat opens a UDP socket for heartbeats. If group is a multicast IP
// the socket joins that group; otherwise it listens for unicast/broadcast.
func listenHeartbeat(port int, group string) (*net.UDPConn, error) {
	if group != "" {
		ip := net.ParseIP(group)
		if ip == nil || !ip.IsMulticast() {
			return nil, fmt.Errorf("invalid multicast group %q", group)
		}
		return net.ListenMulticastUDP("udp4", nil, &net.UDPAddr{IP: ip, Port: port})
	}
	return net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
}

// Watch keeps a heartbeat socket open and calls onHB for each valid heartbeat
// until ctx is cancelled. It is used for continuous discovery so only one
// socket binds the heartbeat port.
func Watch(ctx context.Context, port int, group string, onHB func(*Heartbeat)) error {
	conn, err := listenHeartbeat(port, group)
	if err != nil {
		return fmt.Errorf("listen for heartbeat on udp/%d: %w", port, err)
	}
	defer func() { _ = conn.Close() }()

	go func() {
		<-ctx.Done()
		_ = conn.SetReadDeadline(time.Now())
	}()

	buf := make([]byte, 64*1024)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		hb, perr := parseHeartbeat(buf[:n])
		if perr != nil {
			continue
		}
		if hb.ServerIP == "" && src != nil {
			hb.ServerIP = src.IP.String()
		}
		onHB(hb)
	}
}

// DiscoverOnce listens for a single valid heartbeat and returns it. The server
// IP falls back to the datagram source address when the heartbeat omits it.
func DiscoverOnce(ctx context.Context, port int, group string, timeout time.Duration) (*Heartbeat, error) {
	conn, err := listenHeartbeat(port, group)
	if err != nil {
		return nil, fmt.Errorf("listen for heartbeat on udp/%d: %w", port, err)
	}
	defer func() { _ = conn.Close() }()

	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	_ = conn.SetReadDeadline(deadline)

	// Stop early if the context is cancelled.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.SetReadDeadline(time.Now())
		case <-stop:
		}
	}()

	buf := make([]byte, 64*1024)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			return nil, fmt.Errorf("no heartbeat received: %w", err)
		}
		hb, perr := parseHeartbeat(buf[:n])
		if perr != nil {
			continue // ignore malformed datagrams and keep waiting
		}
		if hb.ServerIP == "" && src != nil {
			hb.ServerIP = src.IP.String()
		}
		return hb, nil
	}
}
