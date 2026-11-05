package bridge

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/macho"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMacToolSelectionAndArchive(t *testing.T) {
	if ytdlpAsset("darwin", "arm64") != "yt-dlp_macos" || ytdlpAsset("darwin", "amd64") != "yt-dlp_macos" || ytdlpAsset("windows", "amd64") != "yt-dlp.exe" || ytdlpAsset("darwin", "386") != "" {
		t.Fatal("wrong yt-dlp asset")
	}
	if len(macFFmpegAssets("arm64")) != 2 || len(macFFmpegAssets("amd64")) != 2 || macFFmpegAssets("386") != nil {
		t.Fatal("wrong FFmpeg assets")
	}
	h := sha256.Sum256([]byte("verified fixture"))
	digest := hex.EncodeToString(h[:])
	for _, mode := range []string{"good", "hash", "traversal", "duplicate", "symlink", "missing"} {
		t.Run(mode, func(t *testing.T) {
			var b bytes.Buffer
			w := zip.NewWriter(&b)
			name := "bin/ffmpeg"
			if mode == "traversal" {
				name = "../ffmpeg"
			}
			if mode == "missing" {
				name = "ffprobe"
			}
			hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
			hdr.SetMode(0755)
			if mode == "symlink" {
				hdr.SetMode(os.ModeSymlink | 0777)
			}
			f, _ := w.CreateHeader(hdr)
			f.Write([]byte("verified fixture"))
			if mode == "duplicate" {
				f, _ = w.Create("other/ffmpeg")
				f.Write([]byte("verified fixture"))
			}
			w.Close()
			dir := t.TempDir()
			archive := filepath.Join(dir, "tool.zip")
			os.WriteFile(archive, b.Bytes(), 0600)
			expected := digest
			if mode == "hash" {
				expected = hex.EncodeToString(make([]byte, 32))
			}
			err := extractMacTool(archive, dir, "ffmpeg", expected)
			if (err == nil) != (mode == "good") {
				t.Fatal(mode, err)
			}
		})
	}
}

// Downloads and checks both architectures without executing Mach-O on Windows.
func TestRealMacFFmpegAssets(t *testing.T) {
	if os.Getenv("VB_INTEGRATION_MAC_ASSETS") != "1" {
		t.Skip("opt-in macOS dependency verification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	i := newToolInstaller(func(message string) { t.Log(message) })
	for _, arch := range []string{"arm64", "amd64"} {
		dir := t.TempDir()
		if _, err := i.macFFmpeg(ctx, dir, arch); err != nil {
			t.Fatal(arch, err)
		}
		for _, name := range []string{"ffmpeg", "ffprobe"} {
			f, err := macho.Open(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			want := macho.CpuAmd64
			if arch == "arm64" {
				want = macho.CpuArm64
			}
			if f.Cpu != want {
				t.Fatal("wrong downloaded architecture", f.Cpu)
			}
			libs, err := f.ImportedLibraries()
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(arch, name, "Mach-O architecture verified; dynamic dependencies:", libs)
		}
		t.Log("Verified macOS archives and pinned executable hashes:", arch)
	}
}

func TestRealMacYTDLPAsset(t *testing.T) {
	if os.Getenv("VB_INTEGRATION_MAC_ASSETS") != "1" {
		t.Skip("opt-in Mac yt-dlp verification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	i := newToolInstaller(nil)
	data, err := i.small(ctx, releaseAPI)
	if err != nil {
		t.Fatal(err)
	}
	var release struct {
		Tag string `json:"tag_name"`
	}
	if json.Unmarshal(data, &release) != nil || versionKey(release.Tag) == "" {
		t.Fatal("invalid release")
	}
	base := "https://github.com/yt-dlp/yt-dlp/releases/download/" + release.Tag + "/"
	sums, err := i.small(ctx, base+"SHA2-256SUMS")
	if err != nil {
		t.Fatal(err)
	}
	digest, err := checksum(sums, "yt-dlp_macos")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "yt-dlp")
	if err = i.verifiedFile(ctx, base+"yt-dlp_macos", file, digest, 64<<20); err != nil {
		t.Fatal(err)
	}
	f, err := macho.OpenFat(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cpus := map[macho.Cpu]bool{}
	for _, a := range f.Arches {
		cpus[a.Cpu] = true
	}
	if !cpus[macho.CpuArm64] || !cpus[macho.CpuAmd64] {
		t.Fatal("yt-dlp is not dual architecture")
	}
	t.Log("Official yt-dlp_macos", release.Tag, "SHA256 and arm64/amd64 slices verified (not executed)")
}
