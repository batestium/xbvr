#!/usr/bin/env bash
# Run inside the development container via dev/run.sh release.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
version="$(cat VERSION)"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] || { echo 'Invalid VERSION' >&2; exit 1; }
git diff --quiet && git diff --cached --quiet || { echo 'Commit tracked changes before a release build.' >&2; exit 1; }
commit="$(git rev-parse HEAD)"
branch="$(git branch --show-current)"
date="$(git show -s --format=%cI HEAD)"
arch="$(go env GOARCH)"
[[ "$(go env GOOS)" == linux ]] || { echo 'This release script targets Linux.' >&2; exit 1; }
release="dist/releases/$version"
staging="$release/linux-$arch"
mkdir -p "$staging"
yarn install --frozen-lockfile
yarn build
go test -mod=readonly -tags=json1 -vet=off ./...
go build -mod=readonly -trimpath -tags=json1 \
  -ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.branch=$branch -X main.date=$date" \
  -o "$staging/xbvr" main.go
cp -a xbvr_data "$staging/"
cat > "$staging/RELEASE.txt" <<INFO
XBVR $version (custom code-match build)
Commit: $commit
Branch: $branch
Commit date: $date
Target: linux/$arch
Toolchain: $(go version)
Built on Debian Bookworm with CGO; requires a compatible glibc.
Stop XBVR and back up its database before replacing the executable.
Keep xbvr_data next to the executable. Rebuild the search index after upgrading.
INFO
name="xbvr-${version}-linux-${arch}"
cp "$staging/xbvr" "$release/$name"
tar -czf "$release/$name.tar.gz" -C "$staging" xbvr xbvr_data RELEASE.txt
(cd "$release" && sha256sum "$name" "$name.tar.gz" > SHA256SUMS)
printf 'Release artifacts: %s\n' "$release"
cat "$release/SHA256SUMS"
