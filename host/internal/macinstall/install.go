// Package macinstall installs only within the invoking user's Library. Keeping
// file operations separate from the CLI permits tests on the build machine.
package macinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const ExtensionID = "knllhbbekgpmakfppghioeknafdonoao"
const hostName = "com.videobridge.host"

func Root(home string) string {
	return filepath.Join(home, "Library", "Application Support", "VideoBridge")
}
func Registrations(home string) []string {
	base := filepath.Join(home, "Library", "Application Support")
	return []string{
		filepath.Join(base, "Google", "Chrome", "NativeMessagingHosts", hostName+".json"),
		filepath.Join(base, "Microsoft Edge", "NativeMessagingHosts", hostName+".json"),
	}
}

func regularBytes(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	return os.ReadFile(path)
}

func directory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe installation directory: %s", path)
	}
	return nil
}

func atomicFile(path string, data []byte, mode fs.FileMode) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe destination: %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".videobridge-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	return err
}

// Verify every package file before replacing any installed component. Reject
// symlinks and unlisted content as well as checksum mismatches.
func Verify(bundle string) error {
	data, err := regularBytes(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		return err
	}
	expected := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		parts := strings.SplitN(strings.TrimSuffix(line, "\r"), "  ", 2)
		if len(parts) != 2 || len(parts[0]) != 64 || strings.Contains(parts[1], "\\") || !fs.ValidPath(parts[1]) {
			return fmt.Errorf("invalid package checksums")
		}
		if _, err := hex.DecodeString(parts[0]); err != nil {
			return err
		}
		if _, ok := expected[parts[1]]; ok {
			return fmt.Errorf("duplicate package file")
		}
		expected[parts[1]] = parts[0]
	}
	err = filepath.WalkDir(bundle, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "SHA256SUMS" {
			return nil
		}
		if entry.Name() == ".DS_Store" {
			return nil
		} // Finder metadata is never copied or executed.
		if _, ok := expected[rel]; !ok {
			return fmt.Errorf("unlisted package file: %s", rel)
		}
		b, err := regularBytes(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(b)
		if hex.EncodeToString(digest[:]) != expected[rel] {
			return fmt.Errorf("package checksum mismatch: %s", rel)
		}
		delete(expected, rel)
		return nil
	})
	if err != nil {
		return err
	}
	if len(expected) != 0 {
		return fmt.Errorf("package files missing")
	}
	return nil
}

type nativeManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Origins     []string `json:"allowed_origins"`
}

func Install(bundle, home, arch string) (string, error) {
	if !filepath.IsAbs(bundle) || !filepath.IsAbs(home) || (arch != "arm64" && arch != "amd64") {
		return "", fmt.Errorf("invalid installation parameters")
	}
	if err := Verify(bundle); err != nil {
		return "", err
	}
	root := Root(home)
	if err := directory(root); err != nil {
		return "", err
	}
	lock, err := lockInstall(filepath.Join(root, ".install.lock"))
	if err != nil {
		return "", fmt.Errorf("another installation may be running: %w", err)
	}
	defer lock.Close()
	hostDir := filepath.Join(root, "host")
	if err := directory(hostDir); err != nil {
		return "", err
	}
	binary, err := regularBytes(filepath.Join(bundle, "host", "darwin-"+arch, "videobridge"))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(binary)
	exe := filepath.Join(hostDir, "videobridge-"+hex.EncodeToString(digest[:])[:12])
	if old, err := regularBytes(exe); err == nil {
		if sha256.Sum256(old) != digest {
			return "", fmt.Errorf("existing Host hash mismatch")
		}
	} else if os.IsNotExist(err) {
		if err = atomicFile(exe, binary, 0700); err != nil {
			return "", err
		}
	} else {
		return "", err
	}
	stage, err := os.MkdirTemp(root, ".install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage) // Only this newly created staging directory.
	next := filepath.Join(stage, "extension")
	if err = directory(next); err != nil {
		return "", err
	}
	source := filepath.Join(bundle, "extension")
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, e := filepath.Rel(source, path)
		if e != nil {
			return e
		}
		target := filepath.Join(next, rel)
		if entry.IsDir() {
			return directory(target)
		}
		if entry.Name() == ".DS_Store" {
			return nil
		}
		data, e := regularBytes(path)
		if e != nil {
			return e
		}
		return atomicFile(target, data, 0600)
	})
	if err != nil {
		return "", err
	}
	config := filepath.Join(hostDir, "config.json")
	if _, err = regularBytes(config); os.IsNotExist(err) {
		cfg := map[string]string{"outputDir": filepath.Join(home, "Downloads", "VideoBridge"), "ytDlpPath": "", "ffmpegPath": ""}
		data, _ := json.MarshalIndent(cfg, "", "  ")
		if err = atomicFile(config, data, 0600); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	current := filepath.Join(root, "extension")
	backup := filepath.Join(stage, "previous-extension")
	hadPrevious := false
	if info, e := os.Lstat(current); e == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsafe extension directory")
		}
		if err = os.Rename(current, backup); err != nil {
			return "", err
		}
		hadPrevious = true
	} else if !os.IsNotExist(e) {
		return "", e
	}
	if err = os.Rename(next, current); err != nil {
		if hadPrevious {
			_ = os.Rename(backup, current)
		}
		return "", err
	}
	for _, file := range Registrations(home) {
		if err = directory(filepath.Dir(file)); err != nil {
			return "", err
		}
		m := nativeManifest{hostName, "VideoBridge local yt-dlp bridge", exe, "stdio", []string{"chrome-extension://" + ExtensionID + "/"}}
		data, _ := json.MarshalIndent(m, "", "  ")
		if err = atomicFile(file, data, 0600); err != nil {
			return "", err
		}
	}
	return exe, nil
}

// Unregister never recursively deletes program files, tools or user downloads.
func Unregister(home string) error {
	if !filepath.IsAbs(home) {
		return fmt.Errorf("home must be absolute")
	}
	for _, file := range Registrations(home) {
		data, err := regularBytes(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var m nativeManifest
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.Name != hostName || filepath.Dir(m.Path) != filepath.Join(Root(home), "host") || !strings.HasPrefix(filepath.Base(m.Path), "videobridge-") {
			continue
		}
		if err = os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}
