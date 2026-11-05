package bridge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func Netscape(w io.Writer, cookies []Cookie) error {
	if _, err := io.WriteString(w, "# Netscape HTTP Cookie File\n# VideoBridge task-local snapshot\n"); err != nil {
		return err
	}
	for _, c := range cookies {
		domain := c.Domain
		sub := "FALSE"
		secure := "FALSE"
		if !c.HostOnly {
			sub = "TRUE"
			if !strings.HasPrefix(domain, ".") {
				domain = "." + domain
			}
		}
		if c.HTTPOnly {
			domain = "#HttpOnly_" + domain
		}
		if c.Secure {
			secure = "TRUE"
		}
		expires := int64(c.ExpirationDate)
		if c.Session {
			expires = 0
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n", domain, sub, c.Path, secure, expires, c.Name, c.Value); err != nil {
			return err
		}
	}
	return nil
}

type CookieFile struct {
	Path, dir string
	lock      *os.File
}

var taskDirName = regexp.MustCompile(`^job-[0-9]+$`)

func PrepareCookieRoot(root string) error {
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid cookie directory")
	}
	if err := restrictDirectory(root); err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || !taskDirName.MatchString(e.Name()) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		lock, err := lockFile(filepath.Join(dir, ".lock"))
		if err != nil {
			continue
		}
		// Live tasks hold an OS lock. Only stale, task-owned directories reach here.
		_ = os.Remove(filepath.Join(dir, "cookies.txt"))
		lock.Close()
		_ = os.Remove(filepath.Join(dir, ".lock"))
		_ = os.Remove(dir)
	}
	return nil
}
func CreateCookieFile(root string, cookies []Cookie) (*CookieFile, error) {
	dir, err := os.MkdirTemp(root, "job-")
	if err != nil {
		return nil, err
	}
	cf := &CookieFile{Path: filepath.Join(dir, "cookies.txt"), dir: dir}
	ok := false
	defer func() {
		if !ok {
			_ = cf.Close()
		}
	}()
	if err := restrictDirectory(dir); err != nil {
		return nil, err
	}
	cf.lock, err = lockFile(filepath.Join(dir, ".lock"))
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(cf.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	err = Netscape(f, cookies)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	ok = true
	return cf, nil
}
func (c *CookieFile) Close() error {
	if c == nil {
		return nil
	}
	err := os.Remove(c.Path)
	if os.IsNotExist(err) {
		err = nil
	}
	if c.lock != nil {
		c.lock.Close()
		c.lock = nil
	}
	_ = os.Remove(filepath.Join(c.dir, ".lock"))
	_ = os.Remove(c.dir)
	return err
}
