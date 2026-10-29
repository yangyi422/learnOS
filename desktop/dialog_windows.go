//go:build windows

package main

import "golang.org/x/sys/windows"

// fatalDialog shows a blocking native message box. The release binary is built
// with -H windowsgui, so without this a startup failure would be invisible.
func fatalDialog(title, message string) {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	messagePtr, _ := windows.UTF16PtrFromString(message)
	const mbIconError = 0x00000010
	_, _ = windows.MessageBox(0, messagePtr, titlePtr, mbIconError|windows.MB_OK)
}
