// Package ids makes the random ids Go-owned tables use.
package ids

import "uuid"

// New returns a random version 4 UUID, the id format the legacy app used.
// The error result preserves the existing caller API; Go's cryptographic
// generator terminates the process if operating-system randomness fails.
func New() (string, error) {
	return uuid.NewV4().String(), nil
}
