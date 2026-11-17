//go:build !windows

package security

import "errors"

var (
	ErrCredentialNotFound = errors.New("credential not found")
	ErrUnsupported        = errors.New("credential store unsupported on this platform")
)

type Credential struct {
	Username string
	Secret   string
}

// ReadGeneric is unavailable outside Windows; the bridge is a Windows tool.
func ReadGeneric(string) (Credential, error) { return Credential{}, ErrUnsupported }