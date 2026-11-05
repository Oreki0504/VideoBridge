#!/bin/bash
set -euo pipefail
cd -- "$(dirname -- "$0")/../host"
if [ "$(uname -s)" != Darwin ]; then echo 'Run this test on macOS.'; exit 1; fi
go test ./...
go vet ./...
go build ./cmd/videobridge
printf '\nGo tests passed on this Mac. Browser installation, actual download, cancellation and Gatekeeper still require the checklist in installer/macos/README-Mac.md.\n'
