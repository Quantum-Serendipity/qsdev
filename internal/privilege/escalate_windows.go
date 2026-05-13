//go:build windows

package privilege

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

func NeedsElevation() bool {
	elevated, err := isWindowsAdmin()
	if err != nil {
		return true
	}
	return !elevated
}

func isWindowsAdmin() (bool, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false, err
	}
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false, err
	}
	return member, nil
}

func DetectElevationTool() string {
	if _, err := exec.LookPath("gsudo"); err == nil {
		return "gsudo"
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		return "sudo"
	}
	return "powershell"
}

func HasCachedCredentials() bool {
	if DetectElevationTool() == "gsudo" {
		return exec.Command("gsudo", "status").Run() == nil
	}
	return false
}
