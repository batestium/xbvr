#!/usr/bin/env bash
set -euo pipefail
repo="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
state="$repo/.local-dev"
mkdir -p "$state/home" "$state/go" "$state/cache" "$state/yarn-cache" "$state/app"
mkdir -p "$repo/dist"
cp -a "$repo/xbvr_data" "$repo/dist/"
mode="${1:-test}"
app_state=app
case "$mode" in build|test|test-all|release) app_state=test-app ;; esac
args=(--rm --init --user "$(id -u):$(id -g)" -v "$repo:/src" -v "$state:/state" -w /src
  -e HOME=/state/home -e GOPATH=/state/go -e GOCACHE=/state/cache
  -e YARN_CACHE_FOLDER=/state/yarn-cache -e XBVR_APPDIR="/state/$app_state" -e DATABASE_URL="sqlite:/state/$app_state/main.db" -e CGO_ENABLED=1)
if [[ -t 0 && -t 1 ]]; then args+=(-it); fi
case "$mode" in
  build)
    command='yarn install --frozen-lockfile && yarn build && go test -mod=readonly -tags=json1 -vet=off ./pkg/api ./pkg/tasks && mkdir -p dist && go build -mod=readonly -tags=json1 -o dist/xbvr main.go'
    ;;
  test)
    command='test -d ui/dist || { echo "Run build first to generate embedded UI assets."; exit 1; }; go test -mod=readonly -tags=json1 -vet=off -v ./pkg/api ./pkg/tasks'
    ;;
  test-all)
    command='test -d ui/dist || { echo "Run build first to generate embedded UI assets."; exit 1; }; go test -mod=readonly -tags=json1 -vet=off ./...'
    ;;
  release)
    command='bash dev/release.sh'
    ;;
  smoke)
    command='node dev/smoke.mjs'
    ;;
  run)
    args+=(--name xbvr-local-dev --stop-timeout 5 -p 127.0.0.1:9999:9999 -p 127.0.0.1:9998:9998)
    command='exec dist/xbvr'
    ;;
  dev)
    args+=(--name xbvr-local-dev --stop-timeout 5 -p 127.0.0.1:9999:9999 -p 127.0.0.1:9998:9998)
    command='go generate && yarn concurrently --kill-others "npm:dev:ui" "air -c dev/air.toml"'
    ;;
  shell)
    exec docker run "${args[@]}" xbvr-dev:local bash
    ;;
  *) echo 'Usage: dev/run.sh {build|test|test-all|release|smoke|run|dev|shell}' >&2; exit 2 ;;
esac
if [[ "$mode" == run || "$mode" == dev ]]; then
  command='mkdir -p "$XBVR_APPDIR/bin"; for tool in ffmpeg ffprobe; do test -e "$XBVR_APPDIR/bin/$tool" || ln -s "/usr/bin/$tool" "$XBVR_APPDIR/bin/$tool"; done; '"$command"
fi
exec docker run "${args[@]}" xbvr-dev:local bash -c "$command"
