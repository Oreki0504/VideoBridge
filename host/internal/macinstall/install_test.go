package macinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func packageFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range map[string]string{
		"host/darwin-arm64/videobridge": "arm executable fixture",
		"host/darwin-amd64/videobridge": "intel executable fixture",
		"extension/manifest.json":       `{"name":"VideoBridge","version":"0.9.0","manifest_version":3}`,
		"extension/chunks/popup.js":     "fixture",
		"Install.command":               "fixture installer",
	} {
		file := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	seal(t, dir)
	return dir
}
func seal(t *testing.T, dir string) {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.IsDir() || entry.Name() == "SHA256SUMS" {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(dir, path)
		if e != nil {
			return e
		}
		h := sha256.Sum256(b)
		lines = append(lines, hex.EncodeToString(h[:])+"  "+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestMacInstallUpgradeAndUnregister(t *testing.T) {
	for _, arch := range []string{"arm64", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			bundle := packageFixture(t)
			home := filepath.Join(t.TempDir(), "用户名 with spaces")
			exe, err := Install(bundle, home, arch)
			if err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(exe); !strings.Contains(string(b), map[string]string{"arm64": "arm", "amd64": "intel"}[arch]) {
				t.Fatal("wrong architecture")
			}
			for _, file := range Registrations(home) {
				b, _ := os.ReadFile(file)
				var m nativeManifest
				if json.Unmarshal(b, &m) != nil || m.Path != exe || len(m.Origins) != 1 || m.Origins[0] != "chrome-extension://"+ExtensionID+"/" {
					t.Fatal("invalid registration", string(b))
				}
			}
			config := filepath.Join(Root(home), "host", "config.json")
			custom := []byte(`{"outputDir":"custom","ytDlpPath":"external","previousYtDlpPath":"previous","ffmpegPath":"ffmpeg"}`)
			os.WriteFile(config, custom, 0600)
			os.WriteFile(filepath.Join(Root(home), "extension", "obsolete.js"), []byte("old"), 0600)
			os.WriteFile(filepath.Join(bundle, "host", "darwin-"+arch, "videobridge"), []byte("new host"), 0700)
			seal(t, bundle)
			next, err := Install(bundle, home, arch)
			if err != nil {
				t.Fatal(err)
			}
			if next == exe {
				t.Fatal("upgrade reused old binary")
			}
			if _, err = os.Stat(exe); err != nil {
				t.Fatal("removed previous host")
			}
			if b, _ := os.ReadFile(config); string(b) != string(custom) {
				t.Fatal("changed user config")
			}
			if _, err = os.Stat(filepath.Join(Root(home), "extension", "obsolete.js")); !os.IsNotExist(err) {
				t.Fatal("obsolete extension file retained")
			}
			// Another installation owns Edge now; do not remove its registration.
			other := nativeManifest{Name: hostName, Path: filepath.Join(home, "other", "host")}
			b, _ := json.Marshal(other)
			os.WriteFile(Registrations(home)[1], b, 0600)
			if err = Unregister(home); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Stat(Registrations(home)[0]); !os.IsNotExist(err) {
				t.Fatal("owned registration retained")
			}
			if _, err = os.Stat(Registrations(home)[1]); err != nil {
				t.Fatal("unrelated registration removed")
			}
			if b, _ := os.ReadFile(config); string(b) != string(custom) {
				t.Fatal("uninstall changed config")
			}
		})
	}
}
func TestMacPackageValidationBeforeMutation(t *testing.T) {
	for _, mode := range []string{"tampered", "missing", "unlisted", "bad-list", "duplicate", "finder"} {
		t.Run(mode, func(t *testing.T) {
			bundle := packageFixture(t)
			home := t.TempDir()
			switch mode {
			case "tampered":
				os.WriteFile(filepath.Join(bundle, "extension", "manifest.json"), []byte("tampered"), 0600)
			case "missing":
				os.Remove(filepath.Join(bundle, "extension", "manifest.json"))
			case "unlisted":
				os.WriteFile(filepath.Join(bundle, "extension", "extra.js"), []byte("extra"), 0600)
			case "bad-list":
				os.WriteFile(filepath.Join(bundle, "SHA256SUMS"), []byte(strings.Repeat("0", 64)+"  ../outside"), 0600)
			case "duplicate":
				b, _ := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
				os.WriteFile(filepath.Join(bundle, "SHA256SUMS"), append(b, b...), 0600)
			case "finder":
				os.WriteFile(filepath.Join(bundle, ".DS_Store"), []byte("Finder metadata"), 0600)
			}
			_, err := Install(bundle, home, "arm64")
			if mode == "finder" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("unsafe package accepted")
			}
			if _, err = os.Stat(Root(home)); !os.IsNotExist(err) {
				t.Fatal("validation mutated installation")
			}
		})
	}
}

func TestMacInstallationLock(t *testing.T) {
	bundle := packageFixture(t)
	home := t.TempDir()
	if err := directory(Root(home)); err != nil {
		t.Fatal(err)
	}
	lock, err := lockInstall(filepath.Join(Root(home), ".install.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err = Install(bundle, home, "arm64"); err == nil {
		t.Fatal("concurrent install accepted")
	}
	if _, err = os.Stat(filepath.Join(Root(home), "host")); !os.IsNotExist(err) {
		t.Fatal("locked install wrote files")
	}
}

func TestDeliveredMacPackage(t *testing.T) {
	bundle := os.Getenv("VB_MAC_PACKAGE_TEST_ROOT")
	if bundle == "" {
		t.Skip("opt-in delivered package file-operation test")
	}
	for _, arch := range []string{"arm64", "amd64"} {
		home := filepath.Join(t.TempDir(), "Mac 用户")
		exe, err := Install(bundle, home, arch)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(exe)
		if err != nil || len(data) < 1000000 {
			t.Fatal("missing delivered executable", err)
		}
		manifest, err := os.ReadFile(filepath.Join(Root(home), "extension", "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var m struct {
			Version string `json:"version"`
			Key     string `json:"key"`
		}
		if json.Unmarshal(manifest, &m) != nil || m.Version != "0.9.0" || m.Key == "" {
			t.Fatal("wrong delivered extension")
		}
		if err = Unregister(home); err != nil {
			t.Fatal(err)
		}
		t.Log(arch, "real package integrity, installation and unregister file operations passed; binary not executed")
	}
}
