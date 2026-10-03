// Package remotesync holds the sync's end-to-end integration test: the real
// confluenceplatform adapter and confluence-client HTTP layer, driven through the real
// planning and execution use cases against an in-process httptest fake Confluence.
//
// It has no production code. It proves, without a real tenant and without writing to any
// real site, that a sync creates the right pages under the right parents, writes the
// annotations back, uploads the attachments, and that a re-run on the same workspace is
// all unchanged — the idempotency the live CI dogfood could never show because it did
// not commit annotations back (#67, #137). It replaced that dogfood (#139).
package remotesync
