# AGENTS.md

## Commands

```bash
make build          # build to bin/gomount
make test           # go test -v -race -cover ./...
make check          # fmt + vet + lint + test (full verification)
go test ./...       # run tests without make
go build ./...      # quick compile check
```

Run `make check` (or `go vet ./...` + `go test ./...`) before committing.

## Architecture

Entry point: `cmd/gomount/main.go` (Cobra CLI).

```
internal/
├── config/         # YAML config loading, types, validation
├── drivers/        # Driver interface + registry + manager
│   ├── smb/        # mount.cifs, credential file based
│   ├── sshfs/      # sshfs, no credential handling (relies on ~/.ssh/config)
│   └── webdav/     # mount.davfs, temp secrets+conf files
├── interaction/    # Terminal I/O, sudo helpers
├── status/         # Mount status detection via moby/sys/mountinfo
└── tui/            # BubbleTea TUI components
```

Drivers implement `drivers.Driver` interface: `Type()`, `Mount()`, `Unmount()`, `Status()`, `Validate()`, `NeedsSudo()`.

## Command Execution Model (important)

All mount/unmount commands run via `interaction.RunCommand(cmd)` (`terminal.go`), NOT via `cmd.CombinedOutput()` or PTY.

`RunCommand` sets:
- `cmd.Stdin = os.Stdin` (real terminal, for interactive prompts like sudo password, mount.davfs credentials)
- `cmd.Stdout = os.Stdout` (real terminal)
- `cmd.Stderr = io.MultiWriter(os.Stderr, &buf)` (visible to user AND captured for error wrapping)

This "hybrid mode" ensures:
- Child processes inherit the real TTY → `isatty(stdin)` is true → interactive prompts work
- sudo timestamp cache binds to real TTY → `EnsureSudoCached()` works across multiple mounts
- No PTY library needed

## Sudo Handling

- Drivers call `interaction.WrapWithSudo(cmd)` internally when `NeedsPrivilege()` is true
- `WrapWithSudo` rewrites `cmd` to `sudo [env VAR=val...] cmd args...`, passing extra env vars from `cmd.Env` via `env` prefix
- `main.go` calls `interaction.EnsureSudoCached()` once before mount loop → user enters sudo password once, subsequent sudo calls hit the cache

## WebDAV Credential Quirk

`mount.davfs` ignores `WEBDAV_USERNAME`/`WEBDAV_PASSWORD` env vars when running as root. WebDAV driver instead creates temp files:
- secrets file: `<URL> <username> <password>`
- conf file: `secrets <path>`
- Passed via `-o conf=<path>` mount option
- Files are `chown root:root` via sudo before mount (mount.davfs requires file owner match)
- Both files cleaned up after mount

## Linux-Only

All underlying commands (`mount.cifs`, `mount.davfs`, `sshfs`, `fusermount`) are Linux-specific. The app targets Linux only.

## Code Style

- No comments unless explicitly requested
- Chinese comments exist in some older code; don't add new ones
- Error wrapping uses `fmt.Errorf` with `%w`
- Driver errors use `drivers.DriverError` struct with Driver/Op/Entry/Err fields
