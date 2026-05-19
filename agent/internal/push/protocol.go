// Package push streams agent snapshots to the remote backend over WebSocket.
package push

import (
	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
)

// ProtocolVersion is bumped whenever the on-the-wire message format changes
// in a non-backward-compatible way. The server must reject messages with an
// unknown version.
const ProtocolVersion = 1

// Envelope wraps every message the agent sends. Keep this tiny — bandwidth
// matters for Cloudflare Worker tail messages.
type Envelope struct {
	V     int              `json:"v"`              // ProtocolVersion
	Type  string           `json:"type"`           // "snapshot" | "hello" | "bye"
	TS    int64            `json:"ts,omitempty"`   // unix milliseconds
	Hello *HelloMessage    `json:"hello,omitempty"`
	Snap  *domain.Snapshot `json:"snap,omitempty"`
}

// HelloMessage is sent once when the WebSocket opens. Identifies the agent
// build and reports the local player's GUID if known, so the backend can
// route to the right user.
type HelloMessage struct {
	AgentVersion string `json:"agentVersion"`
	LocalGuid    string `json:"localGuid,omitempty"`
}
