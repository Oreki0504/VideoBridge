package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type downloadedFile struct {
	Path string `json:"path"`
	ID   string `json:"id"`
}

func finishedName(file downloadedFile, o Options) string {
	base := filepath.Base(file.Path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	marker := " [" + file.ID + "]" + outputSuffix(o)
	// Match only the ID reported by this completed download, at the template's
	// exact suffix position. Brackets belonging to the title must stay intact.
	if file.ID == "" || !strings.HasSuffix(stem, marker) {
		return base
	}
	title := strings.TrimRight(strings.TrimSuffix(stem, marker), " .")
	if title == "" {
		title = "video"
	}
	name := title + outputSuffix(o) + ext
	// Removing the ID can expose a Windows reserved name such as CON.mp4.
	if windowsDeviceName(name) || !filepath.IsLocal(name) {
		name = "_" + name
	}
	if !filepath.IsLocal(name) || filepath.Base(name) != name {
		return base
	}
	return name
}

func windowsDeviceName(name string) bool {
	base := strings.ToUpper(strings.TrimRight(strings.SplitN(name, ".", 2)[0], " "))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	return utf8.RuneCountInString(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && strings.Contains("123456789¹²³", base[3:])
}

func finishFilename(file downloadedFile, o Options) (string, error) {
	name := finishedName(file, o)
	if name == filepath.Base(file.Path) {
		return file.Path, nil
	}
	final, err := publishUnique(file.Path, filepath.Join(filepath.Dir(file.Path), name))
	if err != nil {
		return "", fail("OUTPUT_RENAME_FAILED", "媒体已下载，但整理文件名失败；原文件已保留，请检查文件占用和下载目录。")
	}
	return final, nil
}

// Source and destination stay in the same output volume. publishFile performs
// the no-replace operation atomically, so concurrent downloads cannot overwrite.
func publishUnique(source, final string) (string, error) {
	ext := filepath.Ext(final)
	for n := 1; n <= 10000; n++ {
		target := final
		if n > 1 {
			target = strings.TrimSuffix(final, ext) + fmt.Sprintf(" (%d)", n) + ext
		}
		if err := publishFile(source, target); err == nil {
			// Windows/macOS move the file; other platforms publish a hard link.
			if err := os.Remove(source); err != nil && !os.IsNotExist(err) {
				return "", err
			}
			return target, nil
		} else if !os.IsExist(err) {
			return "", err
		}
	}
	return "", os.ErrExist
}
