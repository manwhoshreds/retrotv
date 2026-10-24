# RetroTV

RetroTV recreates a classic cable television experience from a local media
library. The project has two applications:

- `backend`: a Go server that scans media, builds channel schedules, streams
  content, and serves the configuration web UI.
- `client`: a C++20/Qt 6 appliance interface that discovers the server and
  plays channels through mpv.

## Backend

The backend requires Go, SQLite support supplied by the Go dependencies, and
FFmpeg/ffprobe on `PATH`. From the repository root:

```sh
cd backend
cp .env.example .env
go run ./cmd/retrotv-server serve
```

The web UI is then available at `http://localhost:8080`. Runtime data is stored
under `backend/data/` and is intentionally excluded from Git.

Run the backend test suite with:

```sh
cd backend
go test ./...
```

## Client

The client requires CMake 3.20 or newer, Qt 6, and libmpv:

```sh
cmake -S client -B client/build-current
cmake --build client/build-current
```

See [client/README.md](client/README.md) for the client layout. Local media and
generated guide fixtures belong under `client/testdata/` and should not be
committed.
