package syncexecution

import (
	"crypto/sha256"
	"encoding/hex"
)

// attachmentHash is the hash an annotation records for an attachment. The planner's
// comparison and the write-back both use this one function, so they cannot drift.
func attachmentHash(content []byte) string {
	sum := sha256.Sum256(content)

	return "sha256:" + hex.EncodeToString(sum[:])
}
