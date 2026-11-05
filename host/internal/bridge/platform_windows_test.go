//go:build windows

package bridge

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsCookieACL(t *testing.T) {
	root := t.TempDir()
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	file, err := CreateCookieFile(root, validStart().Cookies)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sd, err := windows.GetNamedSecurityInfo(file.Path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	sddl := sd.String()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sddl, user.User.Sid.String()) || strings.Contains(sddl, ";;;WD)") || strings.Contains(sddl, ";;;BU)") || strings.Contains(sddl, ";;;AU)") {
		t.Fatal("unexpected cookie ACL", sddl)
	}
	if _, err := os.Stat(file.Path); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsCancellationKillsChildren(t *testing.T) {
	tool := buildFixture(t)
	cfg := fixtureConfig(t, tool)
	t.Setenv("VB_FIXTURE_MODE", "cancel-tree")
	root := t.TempDir()
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var child windows.Handle
	_, err := Download(ctx, cfg, root, validStart(), func(_ map[string]any) {
		data, readErr := os.ReadFile(filepath.Join(cfg.OutputDir, "child.pid"))
		if readErr == nil {
			if pid, parseErr := strconv.ParseUint(string(data), 10, 32); parseErr == nil {
				child, _ = windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
			}
		}
		cancel()
	})
	if err != context.Canceled {
		t.Fatalf("unexpected cancellation: %v", err)
	}
	if child == 0 {
		t.Fatal("fixture child was not observed")
	}
	defer windows.CloseHandle(child)
	status, err := windows.WaitForSingleObject(child, 3000)
	if err != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatal("child survived cancellation", status, err)
	}
}
