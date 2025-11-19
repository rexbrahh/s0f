package core

import (
	mathrand "math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(mathrand.New(mathrand.NewSource(time.Now().UnixNano())), 0)
)

// NewNodeID generates a ULID string for new nodes.
func NewNodeID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
