# retrotv

retrotv is a Qt 6 C++ client for simulating a classic cable television experience from owned local media.

This repository starts with the client. Server-side scheduling and media services can be added after the client proves the playback and channel-surfing workflow.

## Stack

- C++20
- Qt 6 Widgets
- mpv integration planned for playback
- CMake

## Build

```sh
cmake -S . -B build
cmake --build build
```

## Layout

- `src/app`: application window and top-level UI
- `src/player`: playback widget and mpv integration
- `src/models`: shared application data models
- `src/m3u`: M3U playlist parsing
- `src/xmltv`: XMLTV guide parsing
- `resources`: visual assets
- `testdata`: local development fixtures
