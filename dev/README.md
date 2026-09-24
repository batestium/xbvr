# Docker development and custom releases

Run these commands from the repository root as your normal Linux/WSL user.
Docker must be available (enable Ubuntu WSL integration in Docker Desktop).

```sh
docker build -f dev/Dockerfile -t xbvr-dev:local dev
./dev/run.sh build
./dev/run.sh dev
```

Open http://localhost:9999/ui/. Go and UI changes rebuild automatically.
Stop with Ctrl+C or `docker stop xbvr-local-dev`. Only localhost ports 9999 and
9998 are published. Containers use the invoking user's UID/GID.

## Commands

- `build`: install locked UI dependencies, build UI, test API/search, build `dist/xbvr`.
- `dev`: watch and rebuild Go/UI; excludes local caches from the Go watcher.
- `run`: run the built development executable without watchers.
- `test`: verbose API and search tests.
- `test-all`: all default Go tests, excluding opt-in live scraper tests.
- `smoke`: start a binary against a disposable DB, check UI/search HTTP responses, stop it.
- `release`: build production UI, run all default tests, embed VERSION/commit/branch/date,
  and produce a stripped Linux executable, tar.gz bundle and SHA256SUMS.
- `shell`: enter a shell with Go, Node, Yarn, Air, GCC and FFmpeg available.

Go commands use CGO and the `json1` tag. Tests use `-vet=off` as documented in
the main README and `-mod=readonly` to keep dependency files unchanged.

## Local data

`.local-dev/` contains caches and isolated development data and is gitignored.
Development DB: `.local-dev/app/main.db`; test DB: `.local-dev/test-app/main.db`.
API integration tests use their own temporary DB and index in a subprocess.
Smoke checks use a disposable directory and never connect to the development DB.
`dist/xbvr_data` is copied from the repository, not symlinked, because XBVR's
startup copy routine does not follow a symlink at the source directory root.

## Release procedure

Set `VERSION` to a custom version, such as `0.4.40-code-match.1`, and commit the
source and version before running `./dev/run.sh release`. The command rejects
uncommitted tracked changes. Outputs are in `dist/releases/<version>/` and are
separate from the executable rewritten by the development watcher.

To test the packaged executable, use a container shell and run:

```sh
XBVR_SMOKE_BINARY=/src/dist/releases/0.4.40-code-match.1/linux-amd64/xbvr node dev/smoke.mjs
```

These builds use Debian Bookworm, the container CPU architecture and glibc.
Check the target system's architecture and glibc before replacing its executable.
No deployment, push or release publication is performed by these scripts.

Back up the application DB, stop the service, replace the binary (keep a copy of
the old one), retain its executable permissions and start the service. Preserve
the existing data directory and service configuration. The archive includes
`xbvr_data`, which belongs next to the binary. Exact service names and paths
must be taken from the target server's configuration.

## Code matching change

Title indexing preserves numbers while keeping lowercase conversion and
punctuation splitting. In Files > Match, a complete four-letter code followed by
numbers retrieves exact title/scene_id matches independently of the 25 full-text
candidates, removes duplicates and ranks exact matches above duration bonuses.
Leading zeroes remain significant. Ordinary queries and field filters keep their
existing behavior. Existing indexes need Reset followed by Rescan to adopt the
new analyzer; exact DB matching also works with an old index.
