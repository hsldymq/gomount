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
├── interaction/    # Terminal I/O, sudo helpers, RunCommand/RunCommandSilent
├── sshtunnel/      # SSH tunnel management (port allocation, state file, lifecycle)
├── status/         # Mount status detection via moby/sys/mountinfo
└── tui/            # BubbleTea TUI (interactive selector) + lipgloss/table (list)
```

Drivers implement `drivers.Driver` interface: `Type()`, `Mount()`, `Unmount()`, `Status()`, `Validate()`, `NeedsSudo()`.

## CLI Structure

```
gomount list|l              # list all entries (plain stdout, lipgloss/table)
gomount interactive|i       # interactive TUI selector (BubbleTea)
gomount mount|m [name...]   # mount named entries (0 args = no-op with hint)
gomount unmount|u [name...] # unmount named entries (0 args = no-op with hint)
gomount config-example|ce   # output example config
```

- `rootCmd` has `SilenceUsage: true` and `SilenceErrors: true` — errors only printed once by `main()`
- Custom help template in `usage.tmpl` with `cmdName` and `cmdNamePadding` template funcs for aligned alias display

## Command Execution Model (important)

Two modes in `interaction/terminal.go`:

**`RunCommand(cmd)`** — for mount operations that may need interactive input (sudo password, mount.davfs credentials, ssh passphrase):
- `cmd.Stdin = os.Stdin` (real terminal)
- `cmd.Stdout = os.Stdout` (real terminal)
- `cmd.Stderr = io.MultiWriter(os.Stderr, &buf)` (visible AND captured)

**`RunCommandSilent(cmd)`** — for unmount operations where stderr should not be shown in real-time:
- `cmd.Stdin = os.Stdin`
- `cmd.Stdout = nil`
- `cmd.Stderr = &buf` (captured only, returned as error message)

Unmount drivers (smb, sshfs, webdav) use `RunCommandSilent` to avoid duplicate stderr output. Error messages are displayed once via the `ERROR:` format in `mountEntries`/`unmountEntries`.

## Shared Mount/Unmount Logic

`mountEntries()` and `unmountEntries()` in `main.go` are the single execution path for both interactive and CLI modes:
- `mountEntries`: already mounted → `Already mounted at: ...`, success++
- `unmountEntries`: not mounted → `Already unmounted`, success++
- Failure → `ERROR: <short message>` (uses `shortError()` to extract innermost error), fail++, continue
- No summary lines printed — callers check `success > 0 || fail > 0` to decide error return

## SSH Tunnel + SMB

`internal/sshtunnel/` manages SSH tunnels for SMB entries that require jump host access:
- Config: `ssh_tunnel.host` field on `MountEntry` (uses `~/.ssh/config` aliases)
- Port allocation: auto from 10500+, via `net.Listen` probe
- Lifecycle: `cmd.Start()` (no `-f` flag) + `waitForPort()` poll, state in `~/.local/share/gomount/tunnels.json`
- Teardown: `fuser -k <port>/tcp` (by port, not PID)
- SMB driver rewrites addr to `localhost:<localPort>` when `SSHTunnel` is set
- `ssh` inherits real stdin via `cmd.Stdin = os.Stdin` for passphrase input

## Sudo Handling

- Drivers call `interaction.WrapWithSudo(cmd)` internally when `NeedsPrivilege()` is true
- `WrapWithSudo` rewrites `cmd` to `sudo [env VAR=val...] cmd args...`
- `mountEntries`/`unmountEntries` call `interaction.EnsureSudoCached()` once per batch

## WebDAV Credential Quirk

`mount.davfs` ignores env vars when running as root. WebDAV driver creates temp files:
- secrets file + conf file, passed via `-o conf=<path>`
- Files are `chown root:root` via sudo, cleaned up after mount

## Linux-Only

All underlying commands (`mount.cifs`, `mount.davfs`, `sshfs`, `fusermount`) are Linux-specific.

## Code Style

- No comments unless explicitly requested
- Chinese comments exist in some older code; don't add new ones
- Error wrapping uses `fmt.Errorf` with `%w`
- Driver errors use `drivers.DriverError` struct with Driver/Op/Entry/Err fields
- Default config: `~/.config/gomount.yaml`
