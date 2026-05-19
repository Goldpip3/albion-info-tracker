// Package push streams agent snapshots to the remote backend over WebSocket.
package push

import (
	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
)

// ProtocolVersion is bumped whenever the on-the-wire message format changes
// in a non-backward-compatible way. The server must reject messages with an
// unknown version.
const ProtocolVersion = 1

// Envelope wraps every message that flows in either direction over the WS.
// The Worker forwards snapshot/hello messages agent→viewers and command
// messages viewer→agent without inspecting the body.
type Envelope struct {
	V       int              `json:"v"`                 // ProtocolVersion
	Type    string           `json:"type"`              // "snapshot" | "hello" | "command"
	TS      int64            `json:"ts,omitempty"`      // unix milliseconds
	Hello   *HelloMessage    `json:"hello,omitempty"`
	Snap    *domain.Snapshot `json:"snap,omitempty"`
	Command *CommandMessage  `json:"command,omitempty"`
}

// HelloMessage is sent once when the WebSocket opens. Identifies the agent
// build and reports the local player's GUID if known.
type HelloMessage struct {
	AgentVersion string `json:"agentVersion"`
	LocalGuid    string `json:"localGuid,omitempty"`
}

// CommandMessage is what a viewer sends to ask the agent to do something
// (currently just session reset). Forwarded transparently by the Worker.
type CommandMessage struct {
	Action string `json:"action"`
}
