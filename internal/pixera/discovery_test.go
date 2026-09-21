package pixera

import (
	"encoding/binary"
	"testing"
)

const sampleHeartbeat = `[
  {"type":"protocol","ip":"","port":1400,"protocol":"JSON/TCP"},
  {"type":"protocol","ip":"","port":8080,"protocol":"HTTP/TCP"},
  {"type":"self","ip":"192.168.10.20"},
  {"type":"liveSystemHeartbeats","heartbeats":[
    {"ip":"192.168.10.20","name":"Local","isLocal":true,"state":"Engine Opened"},
    {"ip":"192.168.10.21","name":"Render-01","isLocal":false,"state":"Engine Opened"}
  ]}
]`

func withHeader(payload []byte) []byte {
	out := append([]byte("pxr1"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(payload)))
	return append(out, payload...)
}

func TestParseHeartbeatWithHeader(t *testing.T) {
	hb, err := parseHeartbeat(withHeader([]byte(sampleHeartbeat)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if hb.APIPort != 1400 {
		t.Fatalf("apiPort = %d, want 1400", hb.APIPort)
	}
	if hb.HTTPPort != 8080 {
		t.Fatalf("httpPort = %d, want 8080", hb.HTTPPort)
	}
	if hb.ServerIP != "192.168.10.20" {
		t.Fatalf("serverIp = %q", hb.ServerIP)
	}
	if len(hb.LiveSystems) != 2 || hb.LiveSystems[1].Name != "Render-01" {
		t.Fatalf("liveSystems = %+v", hb.LiveSystems)
	}
	if got := hb.Target(); got.Host != "192.168.10.20" || got.Port != 1400 {
		t.Fatalf("target = %+v", got)
	}
}

func TestParseHeartbeatWithoutHeader(t *testing.T) {
	hb, err := parseHeartbeat([]byte(sampleHeartbeat))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if hb.APIPort != 1400 {
		t.Fatalf("apiPort = %d", hb.APIPort)
	}
}

func TestParseHeartbeatNoAPIPort(t *testing.T) {
	payload := `[{"type":"self","ip":"10.0.0.1"},{"type":"protocol","port":8080,"protocol":"HTTP/TCP"}]`
	if _, err := parseHeartbeat([]byte(payload)); err == nil {
		t.Fatal("expected error when no JSON/TCP port is advertised")
	}
}
