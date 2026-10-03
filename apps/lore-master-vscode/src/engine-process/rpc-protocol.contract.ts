// The slice of the engine's JSON-RPC contract this process needs. It mirrors the Go
// source of truth in apps/lore-master-engine/rpcprotocol. The full TypeScript mirror
// of every method is #57; this holds only what spawning and health-checking the engine
// requires, and grows no further — other slices own their own contracts.

/** `ping` checks that the engine answers. No params; returns {@link PingResult}. */
export const PING_METHOD = 'ping'

/** The engine's answer to {@link PING_METHOD}. Mirrors Go's `PingResult`. */
export interface PingResult {
  /** Always `"pong"`. */
  pong:    string
  /** The engine's build version. */
  version: string
}
