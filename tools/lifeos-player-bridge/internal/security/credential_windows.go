//go:build windows

// Package security reads the Jellyfin API token from Windows Credential Manager.
// The token never lives in the config file, in a URL, or in a log line.
package security

import (
	"errors"
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	ErrCredentialNotFound = errors.New("credential not found")
	ErrUnsupported        = errors.New("credential store unsupported on this platform")
)

type Credential struct {
	Username string
	Secret   string
}

const credTypeGeneric = 1

type filetime struct {
	Low  uint32
	High uint32
}

type credentialW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32      = syscall.NewLazyDLL("advapi32.dll")
	procCredReadW = advapi32.NewProc("CredReadW")
	procCredFree  = advapi32.NewProc("CredFree")
)

// ReadGeneric loads a generic credential written by cmdkey.
func ReadGeneric(target string) (Credential, error) {
	name, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return Credential{}, fmt.Errorf("credential target: %w", err)
	}
	var block *credentialW
	ok, _, callErr := procCredReadW.Call(uintptr(unsafe.Pointer(name)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&block)))
	if ok == 0 {
		if errno, isErrno := callErr.(syscall.Errno); isErrno && errno == 1168 { // ERROR_NOT_FOUND
			return Credential{}, ErrCredentialNotFound
		}
		return Credential{}, fmt.Errorf("CredRead: %v", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(block)))
	var secret string
	if block.CredentialBlobSize > 0 && block.CredentialBlob != nil {
		secret = DecodeBlob(unsafe.Slice(block.CredentialBlob, block.CredentialBlobSize))
	}
	var username string
	if block.UserName != nil {
		username = utf16PtrToString(block.UserName)
	}
	if secret == "" {
		return Credential{}, ErrCredentialNotFound
	}
	return Credential{Username: username, Secret: secret}, nil
}
// utf16PtrToString decodes a NUL-terminated UTF-16 string without x/sys.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	buffer := make([]uint16, 0, 64)
	for i := 0; i < 4096; i++ {
		value := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i*2)))
		if value == 0 {
			break
		}
		buffer = append(buffer, value)
	}
	return string(utf16.Decode(buffer))
}