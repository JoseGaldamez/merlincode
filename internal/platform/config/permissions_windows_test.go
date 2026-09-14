package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPrivateWindowsACL(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(file, []byte("sentinel"), 0666); err != nil {
		t.Fatal(err)
	}
	// Simulate an inherited or manually granted Everyone ACL on an old file.
	sd, err := windows.SecurityDescriptorFromString("D:(A;;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(file, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateDirectory(dir); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrivateFile(file); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, file} {
		sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatal("DACL inheritance not protected")
		}
		sddl := sd.String()

		userSID := user.User.Sid.String()
		hasCurrentUser := strings.Contains(sddl, ";;;"+userSID+")") ||
			strings.Contains(sddl, ";;;LA)")

		if strings.Count(sddl, "(A;") != 2 ||
			!hasCurrentUser ||
			!strings.Contains(sddl, ";;;SY)") ||
			strings.Contains(sddl, ";;;WD)") {
			t.Fatalf("unexpected DACL: %s", sddl)
		}
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != "sentinel" {
		t.Fatal("permission repair lost data")
	}
}
