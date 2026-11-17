package security

import (
	"strings"
	"unicode/utf16"
)

// DecodeBlob turns a Credential Manager blob into text. cmdkey writes the secret as
// UTF-16LE, PowerShell's ConvertTo-SecureString path may write UTF-8, and both may
// keep a trailing NUL — passing any of that straight into an HTTP header fails with
// `invalid header field value`, so the decoding belongs here.
func DecodeBlob(raw []byte) string {
	for len(raw) > 0 && raw[len(raw)-1] == 0 {
		raw = raw[:len(raw)-1]
	}
	if len(raw) == 0 {
		return ""
	}
	utf16Like := len(raw) > 1
	for i := 1; i < len(raw); i += 2 {
		if raw[i] != 0 {
			utf16Like = false
			break
		}
	}
	if utf16Like {
		units := make([]uint16, 0, (len(raw)+1)/2)
		for i := 0; i < len(raw); i += 2 {
			low := uint16(raw[i])
			var high uint16
			if i+1 < len(raw) {
				high = uint16(raw[i+1])
			}
			units = append(units, low|high<<8)
		}
		return strings.TrimSpace(string(utf16.Decode(units)))
	}
	return strings.TrimSpace(strings.TrimRight(string(raw), "\x00"))
}