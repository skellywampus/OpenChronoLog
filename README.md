# OpenChronoLog

OpenChronoLog is a standalone kiosk-style time clock for small volunteer or workplace operations. It is built with Go and Fyne, stores data in local JSON files, and is intended to run as a self-contained desktop application on macOS and Windows.

## Current Release

This release includes:

- Fullscreen kiosk workflow
- Press `F11` to toggle fullscreen or `Escape` to leave fullscreen
- Badge/ID scan entry
- Duration-only manual entries
- Coordinator lookup and review tools
- Member terminology throughout the interface
- Local JSON storage beside the executable
- ARM Mac and x86 Windows release builds

## Runtime Data Files

The app stores operational data beside the executable:

- `user_database.json`
- `scan_log.json`
- `anomalies_approvals.json`

Do not commit live operational files to GitHub. The `data-template` folder contains empty starter versions that can be copied next to the app when setting up a new installation.

On Windows, keep the JSON files in the same folder as `OpenChronoLog.exe`.

On macOS, keep the JSON files beside the app bundle in the same folder as `OpenChronoLog.app`.

## Build Requirements

- Go
- Fyne dependencies for the target platform
- Docker Desktop for `fyne-cross` Windows builds from macOS
- `fyne-cross` for cross-compiling Windows from macOS

## Build Commands

Run locally:

```sh
go run .
```

Run checks:

```sh
go test ./...
go vet ./...
```

Build ARM Mac:

```sh
make darwin-arm64
```

Build x86 Windows from macOS:

```sh
make windows-amd64
```

## Release Assets

Prebuilt macOS and Windows release files can be attached to a GitHub Release. The source files in this repository do not include generated binaries.
