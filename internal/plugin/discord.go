package plugin

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"chordly/internal/model"
)

const (
	// DefaultDiscordClientID is the default Discord Application Client ID registered for Chordly.
	DefaultDiscordClientID = "121928472918274092"

	// IPC Opcodes
	opHandshake uint32 = 0
	opFrame     uint32 = 1
	opClose     uint32 = 2
	opPing      uint32 = 3
	opPong      uint32 = 4
)

var (
	// ErrNotConnected indicates the client is not connected to a Discord IPC socket.
	ErrNotConnected = errors.New("discord: not connected")
)

// ActivityTimestamps defines start and end playback times in Unix epoch seconds.
type ActivityTimestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// ActivityAssets defines art images and hover text displayed in Discord profile.
type ActivityAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

// DiscordActivity defines the rich presence payload shown in Discord.
type DiscordActivity struct {
	Details    string              `json:"details,omitempty"`
	State      string              `json:"state,omitempty"`
	Timestamps *ActivityTimestamps `json:"timestamps,omitempty"`
	Assets     *ActivityAssets     `json:"assets,omitempty"`
	Instance   bool                `json:"instance"`
}

// DiscordClient handles local IPC communication with Discord's desktop client.
type DiscordClient struct {
	clientID   string
	socketPath string
	dialFunc   func() (net.Conn, error)

	mu        sync.Mutex
	conn      net.Conn
	connected bool
	pid       int
}

// DiscordOption allows customizing DiscordClient behavior.
type DiscordOption func(*DiscordClient)

// WithClientID sets a custom Discord Application ID.
func WithClientID(id string) DiscordOption {
	return func(c *DiscordClient) {
		c.clientID = id
	}
}

// WithSocketPath specifies an explicit IPC socket path (primarily for unit tests).
func WithSocketPath(path string) DiscordOption {
	return func(c *DiscordClient) {
		c.socketPath = path
	}
}

// WithDialer provides a custom connection dialer.
func WithDialer(dial func() (net.Conn, error)) DiscordOption {
	return func(c *DiscordClient) {
		c.dialFunc = dial
	}
}

// NewDiscordClient initializes a new Discord Rich Presence client.
func NewDiscordClient(opts ...DiscordOption) *DiscordClient {
	c := &DiscordClient{
		clientID: DefaultDiscordClientID,
		pid:      os.Getpid(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// findSocketPath locates Discord's IPC UNIX socket across standard Linux paths.
func findSocketPath() (string, error) {
	// Candidate runtime directories
	candidates := []string{
		os.Getenv("XDG_RUNTIME_DIR"),
		os.Getenv("TMPDIR"),
		os.Getenv("TMP"),
		os.Getenv("TEMP"),
		"/tmp",
	}

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		// Discord creates sockets discord-ipc-0 up to discord-ipc-9
		for i := 0; i < 10; i++ {
			sock := filepath.Join(dir, "discord-ipc-"+strconv.Itoa(i))
			if info, err := os.Stat(sock); err == nil && (info.Mode()&os.ModeSocket != 0 || true) {
				return sock, nil
			}
		}
	}

	return "", errors.New("discord: no active IPC socket found")
}

// Connect opens the IPC socket and performs the handshake.
func (c *DiscordClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	var conn net.Conn
	var err error

	if c.dialFunc != nil {
		conn, err = c.dialFunc()
	} else {
		path := c.socketPath
		if path == "" {
			path, err = findSocketPath()
			if err != nil {
				return err
			}
		}
		conn, err = net.DialTimeout("unix", path, 2*time.Second)
	}

	if err != nil {
		return fmt.Errorf("dial discord socket: %w", err)
	}

	c.conn = conn

	// Send handshake frame
	handshake := map[string]any{
		"v":         1,
		"client_id": c.clientID,
	}
	raw, err := json.Marshal(handshake)
	if err != nil {
		conn.Close()
		return fmt.Errorf("marshal handshake: %w", err)
	}

	if err := writeFrame(conn, opHandshake, raw); err != nil {
		conn.Close()
		return fmt.Errorf("write handshake frame: %w", err)
	}

	// Read handshake response
	respOp, _, err := readFrame(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("read handshake response: %w", err)
	}
	if respOp != opFrame {
		conn.Close()
		return fmt.Errorf("unexpected handshake response opcode: %d", respOp)
	}

	c.connected = true
	return nil
}

// Close disconnects from the Discord IPC socket.
func (c *DiscordClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	if c.conn != nil {
		_ = writeFrame(c.conn, opClose, []byte("{}"))
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// IsConnected returns whether the client is currently connected.
func (c *DiscordClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// SetActivity updates the rich presence activity in Discord.
func (c *DiscordClient) SetActivity(act *DiscordActivity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return ErrNotConnected
	}

	payload := map[string]any{
		"cmd":   "SET_ACTIVITY",
		"nonce": uuid.New().String(),
		"args": map[string]any{
			"pid":      c.pid,
			"activity": act,
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal set_activity: %w", err)
	}

	if err := writeFrame(c.conn, opFrame, raw); err != nil {
		c.connected = false
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("write set_activity frame: %w", err)
	}

	// Read response (non-blocking verification)
	_, _, err = readFrame(c.conn)
	if err != nil {
		c.connected = false
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("read set_activity response: %w", err)
	}

	return nil
}

// ClearActivity removes any current rich presence activity.
func (c *DiscordClient) ClearActivity() error {
	return c.SetActivity(nil)
}

// UpdateTrack adapts Chordly track and playback state into a Discord activity.
func (c *DiscordClient) UpdateTrack(track model.Track, status model.PlaybackState, playhead time.Duration) error {
	if track.Title == "" {
		return c.ClearActivity()
	}

	act := &DiscordActivity{
		Details: track.Title,
		State:   track.Artist,
		Assets: &ActivityAssets{
			LargeImage: "chordly_logo",
			LargeText:  track.Title,
		},
	}

	if status == model.StatePlaying {
		now := time.Now().Unix()
		elapsedSec := int64(playhead.Seconds())
		startTime := now - elapsedSec
		act.Timestamps = &ActivityTimestamps{
			Start: startTime,
		}
		if track.Duration > 0 {
			act.Timestamps.End = startTime + int64(track.Duration.Seconds())
		}
		act.Assets.SmallImage = "play"
		act.Assets.SmallText = "Playing"
	} else {
		act.Assets.SmallImage = "pause"
		act.Assets.SmallText = "Paused"
	}

	return c.SetActivity(act)
}

// writeFrame serializes a Discord IPC binary frame: 4 bytes opcode + 4 bytes length + payload bytes.
func writeFrame(w io.Writer, opcode uint32, payload []byte) error {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], opcode)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))

	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// readFrame deserializes a Discord IPC binary frame.
func readFrame(r io.Reader) (uint32, []byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}

	return opcode, payload, nil
}
