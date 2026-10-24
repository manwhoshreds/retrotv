#!/usr/bin/env bash
# Sync the client to the Pi, build it there, restart it and follow its log.
#
# Usage: scripts/deploy-pi.sh [--no-log | --stop]
#
#   (none)    deploy and follow the log; Ctrl+C stops following and stops RetroTV
#   --no-log  deploy and leave RetroTV running without following the log
#   --stop    stop the running client on the Pi and exit without deploying
#
# Environment overrides:
#   PI_HOST          ssh host of the Pi (default: 192.168.0.79)
#   PI_DIR           checkout directory on the Pi, relative to $HOME (default: retrotv-dev)
#   QT_QPA_PLATFORM  force a Qt platform on the Pi instead of auto-detecting (e.g. eglfs)
#   RETROTV_ENV      extra space-separated VAR=value pairs to export for the client
set -euo pipefail

PI_HOST="${PI_HOST:-192.168.0.79}"
PI_DIR="${PI_DIR:-retrotv-dev}"
FOLLOW_LOG=1

stop_retrotv() {
    local status=0
    ssh "$PI_HOST" "pkill -x retrotv" || status=$?
    case "$status" in
        0) echo "RetroTV stopped on $PI_HOST" ;;
        1) echo "RetroTV was not running on $PI_HOST" ;;
        *) echo "Could not stop RetroTV on $PI_HOST (exit $status)" >&2; return "$status" ;;
    esac
}

case "${1:-}" in
    --no-log)
        FOLLOW_LOG=0
        ;;
    --stop)
        stop_retrotv
        exit
        ;;
esac

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Syncing client to $PI_HOST:~/$PI_DIR/client"
ssh "$PI_HOST" "mkdir -p ~/$PI_DIR"
# Excluded paths are not deleted on the Pi, so its build directory survives --delete.
rsync -az --delete \
    --exclude 'build/' \
    --exclude 'build-*/' \
    --exclude 'testdata/' \
    "$REPO_ROOT/client/" "$PI_HOST:$PI_DIR/client/"

echo "==> Building and restarting on $PI_HOST"
ssh "$PI_HOST" "PI_DIR='$PI_DIR' QT_QPA_PLATFORM='${QT_QPA_PLATFORM:-}' RETROTV_ENV='${RETROTV_ENV:-}' bash -s" <<'REMOTE'
set -euo pipefail
cd "$HOME/$PI_DIR/client"

if [[ ! -f build/CMakeCache.txt ]]; then
    generator=()
    launcher=()
    command -v ninja >/dev/null && generator=(-G Ninja)
    command -v ccache >/dev/null && launcher=(-DCMAKE_CXX_COMPILER_LAUNCHER=ccache)
    cmake -S . -B build "${generator[@]}" "${launcher[@]}" -DCMAKE_BUILD_TYPE=RelWithDebInfo
fi
cmake --build build --parallel "$(nproc)"

pkill -x retrotv && sleep 1 || true

# Attach to whatever display the Pi has: a desktop session's Wayland or X11
# server, or the bare framebuffer via eglfs when no desktop is running.
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
if [[ -z "$QT_QPA_PLATFORM" ]]; then
    unset QT_QPA_PLATFORM
    wayland_socket="$(find "$XDG_RUNTIME_DIR" -maxdepth 1 -name 'wayland-[0-9]' -printf '%f\n' 2>/dev/null | head -n1)"
    if [[ -n "$wayland_socket" ]]; then
        export WAYLAND_DISPLAY="$wayland_socket"
    elif [[ -S /tmp/.X11-unix/X0 ]]; then
        export DISPLAY=:0
    else
        export QT_QPA_PLATFORM=eglfs
    fi
fi

# eglfs otherwise picks the TV's preferred mode, which on a 4K set leaves the
# Pi's GPU too slow to composite every video frame.
if [[ "${QT_QPA_PLATFORM:-}" == eglfs && -z "${QT_QPA_EGLFS_KMS_CONFIG:-}" ]]; then
    export QT_QPA_EGLFS_KMS_CONFIG="$PWD/eglfs-kms.json"
fi

if [[ -n "$RETROTV_ENV" ]]; then
    # Word splitting is intended: each VAR=value pair becomes its own export.
    # shellcheck disable=SC2086
    export $RETROTV_ENV
fi

setsid nohup build/retrotv >"$HOME/$PI_DIR/retrotv.log" 2>&1 </dev/null &
echo "RetroTV started (pid $!), log: ~/$PI_DIR/retrotv.log"
REMOTE

if [[ "$FOLLOW_LOG" == 1 ]]; then
    echo "==> Following log (Ctrl+C stops RetroTV)"
    # Ctrl+C ends the remote tail; keep this script alive through it (a no-op
    # handler, unlike an ignored signal, is not inherited by ssh) so the
    # client is stopped once the log stream closes.
    trap ':' INT
    ssh -t "$PI_HOST" "tail -n 50 -F ~/$PI_DIR/retrotv.log" || true
    trap - INT
    echo
    stop_retrotv
fi
