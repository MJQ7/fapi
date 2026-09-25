package core

import (
	"crypto/rand"
	"fmt"
)

// newID returns a random ID in the UUID format, such as
// "7e8a6c49-daf8-43e2-86dd-a691c844f176".
func newID() string {
	var bytes [16]byte
	// rand.Read never returns an error (it crashes the program instead, as
	// the Go documentation explains), so there's nothing to handle.
	_, _ = rand.Read(bytes[:])

	// Mark it as a version 4 (random) UUID.
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}
