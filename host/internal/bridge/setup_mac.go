package bridge

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Pinned executable hashes published at https://www.osxexperts.net/ on
// 2026-11-04. These third-party static builds are downloaded, never bundled.
// A changed upstream asset fails closed until the pin is reviewed.
type macToolAsset struct{ name, url, digest string }

func macFFmpegAssets(arch string) []macToolAsset {
	switch arch {
	case "arm64":
		return []macToolAsset{
			{"ffmpeg", "https://www.osxexperts.net/ffmpeg902arm.zip", "762f16b2913b8d3b8fdb8092df12b691053b035b13ae9b88205513c55cb8e71e"},
			{"ffprobe", "https://www.osxexperts.net/ffprobe902arm.zip", "e9d3ad989b11d1ff4e3b0d140df170a6b38f194f83ac77f0e05a127a0f3a0ae4"},
		}
	case "amd64":
		return []macToolAsset{
			{"ffmpeg", "https://www.osxexperts.net/ffmpeg80intel.zip", "df3f1e3facdc1ae0ad0bd898cdfb072fbc9641bf47b11f172844525a05db8d11"},
			{"ffprobe", "https://www.osxexperts.net/ffprobe80intel.zip", "5228e651e2bd67bb55819b27f6138351587b16d2b87446007bf35b7cf930d891"},
		}
	}
	return nil
}

func extractMacTool(archive, dest, name, digest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	found := false
	for _, f := range r.File {
		if path.Base(f.Name) != name {
			continue
		}
		if found || f.Mode()&os.ModeType != 0 || strings.Contains(f.Name, "\\") || strings.HasPrefix(f.Name, "/") || strings.Contains("/"+f.Name+"/", "/../") || f.UncompressedSize64 > 300<<20 {
			return fmt.Errorf("unsafe macOS tool archive")
		}
		found = true
		src, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(filepath.Join(dest, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			src.Close()
			return err
		}
		hash := sha256.New()
		n, err := io.Copy(io.MultiWriter(out, hash), io.LimitReader(src, (300<<20)+1))
		src.Close()
		ce := out.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		if n > 300<<20 || hex.EncodeToString(hash.Sum(nil)) != digest {
			return fmt.Errorf("macOS tool SHA256 mismatch")
		}
		if err = os.Chmod(filepath.Join(dest, name), 0700); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("macOS tool missing")
	}
	return nil
}

func (i toolInstaller) macFFmpeg(ctx context.Context, stage, arch string) (string, error) {
	assets := macFFmpegAssets(arch)
	if len(assets) != 2 {
		return "", fmt.Errorf("unsupported macOS architecture")
	}
	for _, a := range assets {
		i.progress("下载并校验 macOS " + a.name + "（OSXExperts 构建）…")
		archive := filepath.Join(stage, a.name+".zip")
		f, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return "", err
		}
		err = i.fetch(ctx, a.url, f, 150<<20)
		ce := f.Close()
		if err == nil {
			err = ce
		}
		if err == nil {
			err = extractMacTool(archive, stage, a.name, a.digest)
		}
		os.Remove(archive)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(stage, "ffmpeg"), nil
}
