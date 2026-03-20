package decoy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// SystemDirective is appended to the first system message (or prepended as a new system message).
const systemDirectiveFmt = "\n[decoy_session id=%s integrity=sha256-placeholder]"

// NewID returns an unpredictable decoy token for this request.
func NewID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "decoy-" + hex.EncodeToString(b[:]), nil
}

// InjectIntoSystem returns updated system content with a decoy marker.
func InjectIntoSystem(systemContent, decoyID string) string {
	return systemContent + fmt.Sprintf(systemDirectiveFmt, decoyID)
}
