#!/usr/bin/env bash
# Build the backend, copy it to the server machine, restart it and follow its log.
#
# Usage: scripts/deploy-backend.sh [--no-log | --stop | --log]
#
#   (none)    deploy and follow the log; Ctrl+C stops following, RetroTV keeps running
#   --no-log  deploy without following the log
#   --stop    stop the backend on the server and exit without deploying
#   --log     follow the server's log without deploying
#
# Environment:
#   BACKEND_HOST  ssh host of the server machine (default: 192.168.0.14)
#
# Everything lands in ~/retrotv on the server, run by the systemd user unit in
# scripts/retrotv.service. An existing ~/retrotv/.env and ~/retrotv/data/ are
# never overwritten; on the first deploy .env is created from .env.example.
set -euo pipefail

BACKEND_HOST="${BACKEND_HOST:-192.168.0.14}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REMOTE_DIR=retrotv   # must match WorkingDirectory in retrotv.service
FOLLOW_LOG=1

follow_log() {
    echo "==> Following log on $BACKEND_HOST (Ctrl+C stops following; RetroTV keeps running)"
    ssh -t "$BACKEND_HOST" "journalctl --user -u retrotv -n 50 -f" || true
}

case "${1:-}" in
    "") ;;
    --no-log)
        FOLLOW_LOG=0
        ;;
    --stop)
        ssh "$BACKEND_HOST" "systemctl --user stop retrotv"
        echo "RetroTV backend stopped on $BACKEND_HOST"
        exit
        ;;
    --log)
        follow_log
        exit
        ;;
    *)
        echo "Unknown option: $1" >&2
        exit 1
        ;;
esac

echo "==> Checking $BACKEND_HOST"
remote_arch="$(ssh "$BACKEND_HOST" "uname -m")"
case "$remote_arch" in
    x86_64)        goarch=amd64 ;;
    aarch64|arm64) goarch=arm64 ;;
    *) echo "Unsupported server architecture: $remote_arch" >&2; exit 1 ;;
esac
ssh "$BACKEND_HOST" "command -v ffmpeg >/dev/null && command -v ffprobe >/dev/null" \
    || echo "warning: ffmpeg/ffprobe not found on $BACKEND_HOST; install ffmpeg before streaming" >&2

binary="$REPO_ROOT/backend/dist/retrotv-server-linux-$goarch"
echo "==> Building backend for linux/$goarch"
(cd "$REPO_ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
    go build -o "$binary" ./cmd/retrotv-server)

echo "==> Copying to $BACKEND_HOST:~/$REMOTE_DIR"
ssh "$BACKEND_HOST" "mkdir -p ~/$REMOTE_DIR/data ~/.config/systemd/user"
# rsync writes to a temp file and renames it, so replacing the running binary is safe.
rsync -az "$binary" "$BACKEND_HOST:$REMOTE_DIR/retrotv-server"
rsync -az "$REPO_ROOT/scripts/retrotv.service" "$BACKEND_HOST:.config/systemd/user/retrotv.service"
rsync -az --ignore-existing "$REPO_ROOT/backend/.env.example" "$BACKEND_HOST:$REMOTE_DIR/.env"

echo "==> Restarting RetroTV on $BACKEND_HOST"
ssh "$BACKEND_HOST" bash -s <<'REMOTE'
set -euo pipefail
systemctl --user daemon-reload
systemctl --user enable --quiet retrotv
systemctl --user restart retrotv
sleep 1
systemctl --user is-active --quiet retrotv \
    || { echo "RetroTV failed to start:" >&2; journalctl --user -u retrotv -n 20 --no-pager >&2; exit 1; }
echo "RetroTV running on $(hostname)"
if [[ "$(loginctl show-user "$USER" -p Linger --value 2>/dev/null)" != yes ]]; then
    echo "note: lingering is off, so RetroTV stops when you log out and won't start at boot."
    echo "      Fix once on this machine with: sudo loginctl enable-linger $USER"
fi
REMOTE

if [[ "$FOLLOW_LOG" == 1 ]]; then
    follow_log
fi
