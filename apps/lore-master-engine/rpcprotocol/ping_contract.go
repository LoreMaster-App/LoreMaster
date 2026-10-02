package rpcprotocol

// MethodPing checks that the engine answers. Params: none. Result: PingResult.
const MethodPing = "ping"

// PingResult answers ping.
type PingResult struct {
	// Pong is always "pong".
	Pong string `json:"pong"`
	// Version is the engine's build version.
	Version string `json:"version"`
}
