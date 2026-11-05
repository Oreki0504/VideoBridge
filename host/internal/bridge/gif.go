package bridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ConvertGIF(ctx context.Context, ffmpeg, input, outputDir string, o Options) (string, error) {
	name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	suffix := fmt.Sprintf(" [gif-%ds-%ds-%dw-%dfps].gif", o.GIFStart, o.gifDuration(), o.gifWidth(), o.gifFPS())
	final := filepath.Join(outputDir, name+suffix)
	// Work inside the task-owned source directory. Failed/cancelled conversions
	// never expose a partial GIF as a successful result.
	temporary := filepath.Join(filepath.Dir(input), "converted.gif")
	filter := fmt.Sprintf("fps=%d,scale='min(%d,iw)':-1:flags=lanczos,split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=sierra2_4a", o.gifFPS(), o.gifWidth())
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-ss", strconv.Itoa(o.GIFStart), "-i", input, "-t", strconv.Itoa(o.gifDuration()), "-an", "-filter_complex", filter, "-loop", "0", "-y", temporary)
	cmd.WaitDelay = 3 * time.Second
	// Raw ffmpeg output can contain input paths; return only structured failures.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := runCommand(cmd); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fail("GIF_CONVERSION_FAILED", "GIF 转换失败，请检查时间范围与 FFmpeg。")
	}
	info, err := os.Stat(temporary)
	if err != nil || info.Size() < 14 {
		return "", fail("GIF_EMPTY", "所选时间范围没有可转换的视频帧。")
	}
	file, err := os.Open(temporary)
	if err != nil {
		return "", fail("GIF_CONVERSION_FAILED", "无法验证 GIF 输出。")
	}
	var header [6]byte
	_, err = io.ReadFull(file, header[:])
	file.Close()
	if err != nil || (string(header[:]) != "GIF89a" && string(header[:]) != "GIF87a") {
		return "", fail("GIF_CONVERSION_FAILED", "输出不是有效的 GIF。")
	}
	// Publish the completed file without replacing an existing user file.
	// Repeated requests get a new numbered result.
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if target, err := publishUnique(temporary, final); err == nil {
		return target, nil
	} else if !os.IsExist(err) {
		return "", fail("OUTPUT_UNAVAILABLE", "无法保存 GIF，请检查下载目录及磁盘文件系统。")
	}
	return "", fail("OUTPUT_EXISTS", "同名 GIF 文件过多，请整理输出目录。")
}
