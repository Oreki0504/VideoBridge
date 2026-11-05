# macOS dependency sources

- yt-dlp: https://github.com/yt-dlp/yt-dlp and https://github.com/yt-dlp/yt-dlp-nightly-builds — official `yt-dlp_macos`, checksum from `SHA2-256SUMS`.
- FFmpeg and FFprobe: https://www.osxexperts.net/ — third-party static builds, not official FFmpeg binaries. The author labels them for educational purposes; review the upstream terms for your use. Binaries are downloaded on the user's machine, not redistributed in this package. Existing Homebrew or manually configured tools are preferred.
- Pinned hashes were reviewed on 2026-11-04. A changed file fails verification and is never executed. Version changes require a reviewed VideoBridge release; FFmpeg does not self-update.
- Go Host uses golang.org/x/sys and golang.org/x/net (BSD-3-Clause). Browser bundle includes tldts / tldts-core (MIT). Third-party license texts are included in THIRD-PARTY-LICENSES.txt.

| Architecture | Asset | Executable SHA256 |
| --- | --- | --- |
| Apple Silicon | https://www.osxexperts.net/ffmpeg902arm.zip | 762f16b2913b8d3b8fdb8092df12b691053b035b13ae9b88205513c55cb8e71e |
| Apple Silicon | https://www.osxexperts.net/ffprobe902arm.zip | e9d3ad989b11d1ff4e3b0d140df170a6b38f194f83ac77f0e05a127a0f3a0ae4 |
| Intel | https://www.osxexperts.net/ffmpeg80intel.zip | df3f1e3facdc1ae0ad0bd898cdfb072fbc9641bf47b11f172844525a05db8d11 |
| Intel | https://www.osxexperts.net/ffprobe80intel.zip | 5228e651e2bd67bb55819b27f6138351587b16d2b87446007bf35b7cf930d891 |
