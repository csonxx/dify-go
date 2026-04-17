package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("generate id: %w", err))
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
