package plugin

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
	"time"

	"chordly/internal/model"
)

func TestFrameSerialization(t *testing.T) {
	buf := new(bytes.Buffer)
	payload := []byte(`{"test": "frame"}`)

	err := writeFrame(buf, opFrame, payload)
	if err != nil {
		t.Fatalf("writeFrame failed: %v", err)
	}

	opcode, readPayload, err := readFrame(buf)
	if err != nil {
		t.Fatalf("readFrame failed: %v", err)
	}

	if opcode != opFrame {
		t.Errorf("expected opcode %d, got %d", opFrame, opcode)
	}
	if string(readPayload) != string(payload) {
		t.Errorf("expected payload %s, got %s", payload, readPayload)
	}
}

func startMockDiscordIPC(t *testing.T) (net.Listener, chan map[string]any) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	activityCh := make(chan map[string]any, 10)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()

				// 1. Expect handshake
				op, data, err := readFrame(c)
				if err != nil || op != opHandshake {
					return
				}
				_ = data

				// Respond with READY
				readyResp := []byte(`{"cmd":"DISPATCH","evt":"READY"}`)
				_ = writeFrame(c, opFrame, readyResp)

				// 2. Handle commands
				for {
					cmdOp, cmdData, err := readFrame(c)
					if err != nil {
						return
					}
					if cmdOp == opClose {
						return
					}

					var parsed map[string]any
					if err := json.Unmarshal(cmdData, &parsed); err == nil {
						activityCh <- parsed
					}

					// Acknowledge command
					ack := []byte(`{"cmd":"SET_ACTIVITY","data":{}}`)
					_ = writeFrame(c, opFrame, ack)
				}
			}(conn)
		}
	}()

	return l, activityCh
}

func TestDiscordClientLiveFlow(t *testing.T) {
	listener, activityCh := startMockDiscordIPC(t)
	defer listener.Close()

	addr := listener.Addr().String()

	client := NewDiscordClient(
		WithClientID("999999999"),
		WithDialer(func() (net.Conn, error) {
			return net.DialTimeout("tcp", addr, 2*time.Second)
		}),
	)

	// Verify Connect
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !client.IsConnected() {
		t.Errorf("expected IsConnected to be true")
	}

	// UpdateTrack Playing
	track := model.Track{
		ID:       "track-1",
		Title:    "Starboy",
		Artist:   "The Weeknd",
		Duration: 230 * time.Second,
	}

	playhead := 45 * time.Second
	err := client.UpdateTrack(track, model.StatePlaying, playhead)
	if err != nil {
		t.Fatalf("UpdateTrack failed: %v", err)
	}

	select {
	case payload := <-activityCh:
		cmd, _ := payload["cmd"].(string)
		if cmd != "SET_ACTIVITY" {
			t.Errorf("expected SET_ACTIVITY, got %s", cmd)
		}
		args, _ := payload["args"].(map[string]any)
		activity, _ := args["activity"].(map[string]any)
		if activity["details"] != "Starboy" {
			t.Errorf("expected details 'Starboy', got %v", activity["details"])
		}
		if activity["state"] != "The Weeknd" {
			t.Errorf("expected state 'The Weeknd', got %v", activity["state"])
		}
		assets, _ := activity["assets"].(map[string]any)
		if assets["small_text"] != "Playing" {
			t.Errorf("expected small_text 'Playing', got %v", assets["small_text"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for activity payload")
	}

	// UpdateTrack Paused
	err = client.UpdateTrack(track, model.StatePaused, playhead)
	if err != nil {
		t.Fatalf("UpdateTrack paused failed: %v", err)
	}

	select {
	case payload := <-activityCh:
		args, _ := payload["args"].(map[string]any)
		activity, _ := args["activity"].(map[string]any)
		assets, _ := activity["assets"].(map[string]any)
		if assets["small_text"] != "Paused" {
			t.Errorf("expected small_text 'Paused', got %v", assets["small_text"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for paused activity")
	}

	// ClearActivity
	err = client.ClearActivity()
	if err != nil {
		t.Fatalf("ClearActivity failed: %v", err)
	}

	select {
	case payload := <-activityCh:
		args, _ := payload["args"].(map[string]any)
		if args["activity"] != nil {
			t.Errorf("expected nil activity on clear, got %v", args["activity"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for clear activity")
	}

	// Close
	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if client.IsConnected() {
		t.Errorf("expected IsConnected to be false after close")
	}
}

func TestDiscordClientNotConnectedError(t *testing.T) {
	client := NewDiscordClient()
	err := client.SetActivity(&DiscordActivity{Details: "Testing"})
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}
