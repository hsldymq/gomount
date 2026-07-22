# gomount

A convenient CLI tool for managing SMB/CIFS, SSHFS, WebDAV, and Alibaba Cloud OSS mounts with an interactive TUI.

## Features

- **Simple Configuration**: Define all your mounts in a single YAML file
- **Multiple Protocols**: Supports SMB/CIFS, SSHFS, WebDAV, and Alibaba Cloud OSS
- **Interactive TUI**: Beautiful terminal UI for browsing and selecting shares
- **Mount Status Tracking**: Check which shares are currently mounted
- **Interactive Selection**: Easy selection for mount/unmount operations
- **Password Prompting**: Secure password input with visual feedback
- **Privilege Escalation**: Automatic sudo handling when needed
- **Mount Point Management**: Automatic directory creation and cleanup

## Installation

### From Source

```bash
git clone https://github.com/hsldymq/gomount.git
cd gomount
make install
```

### Using Go

For Linux

```sh
go install github.com/hsldymq/gomount/cmd/gomount@latest
```

For MacOS
```sh
CGO_ENABLED=1 go install -tags cmount github.com/hsldymq/gomount/cmd/gomount@latest
```

### Manual Installation

```bash
git clone https://github.com/hsldymq/gomount.git
cd gomount
make build
sudo cp bin/gomount /usr/local/bin/
```

## Requirements

- Linux or macOS operating system
- Linux: `mount.cifs` command (install `cifs-utils` package) — for SMB mounts
- macOS: `mount_smbfs` command — for SMB mounts
- `sshfs` command — for SSHFS mounts
- Linux cloud mounts require FUSE support
- macOS cloud mounts require macFUSE, CGO, and a binary built with `-tags cmount`
- Linux SMB mounts require `sudo` access
- Go 1.25+ (for building from source)

For WebDAV and OSS mounts on macOS, install macFUSE and build the macOS-enabled binary:

```bash
brew install --cask macfuse
CGO_ENABLED=1 go build -tags cmount -o bin/gomount ./cmd/gomount
```

During development use `CGO_ENABLED=1 go run -tags cmount ./cmd/gomount i`. A regular macOS build still supports SMB and SSHFS, but cloud mounts return an actionable build error.

### Install Dependencies

On Debian/Ubuntu:
```bash
sudo apt-get install cifs-utils sshfs
```

On Fedora/RHEL:
```bash
sudo dnf install cifs-utils fuse-sshfs
```

On Arch Linux:
```bash
sudo pacman -S cifs-utils sshfs
```

## Configuration

Create a configuration file at `~/.config/gomount.yaml`:

```yaml
daemon:
  log_target: syslog                 # syslog, file, or stderr; default syslog
  # log_file: ~/.local/share/gomount/gomount-daemon.log
  # socket_path: ~/.local/share/gomount/gomount.sock

mounts:
  # SMB/CIFS mount
  - name: nas
    type: smb
    smb:
      addr: 192.168.1.100
      # port: 445                   # optional, default 445
      share_name: shared_folder
      username: user
      # password: pass              # optional, prompts interactively if omitted
    mount_dir_path: /mnt/nas

  # SSHFS mount
  # Connection details (username, port, keys, etc.) are managed by ~/.ssh/config
  - name: dev-server
    type: sshfs
    sshfs:
      host: dev.example.com         # ~/.ssh/config alias or hostname
      remote_path: /home/user/projects
    mount_dir_path: /mnt/dev

  # WebDAV mount (managed by gomount daemon)
  - name: docs
    type: webdav
    webdav:
      url: https://cloud.example.com/remote.php/dav/files/user/
      username: user                # optional
      password: pass                # optional
      path: /team/docs              # optional path under the WebDAV endpoint
    mount_dir_path: ~/Mounts/docs

  # Alibaba Cloud OSS mount (managed by gomount daemon)
  - name: aliyun-archive
    type: aliyun_oss
    aliyun_oss:
      bucket: my-bucket
      path: backups
      endpoint: oss-cn-hangzhou.aliyuncs.com
      access_key_id: your-access-key-id
      access_key_secret: your-access-key-secret
      # security_token: your-sts-token
    mount_dir_path: ~/Mounts/aliyun-archive

```

You can generate a full example config with:

```bash
gomount config-example > ~/.config/gomount.yaml
```

### Configuration Options

#### Common Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Unique identifier for this mount |
| `type` | Yes | - | Mount type (`smb`, `sshfs`, `webdav`, `aliyun_oss`). |
| `mount_dir_path` | Yes | - | Full local path for the mount point. Supports `~` expansion. |

#### Daemon (`daemon:` block)

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `daemon.log_target` | No | `syslog` | Daemon output target. `syslog` falls back to `stderr` if unavailable. Use `file` for a fixed log file or `stderr` for debugging. |
| `daemon.log_file` | No | `~/.local/share/gomount/gomount-daemon.log` | Log file used when `log_target` is `file`. |
| `daemon.socket_path` | No | `~/.local/share/gomount/gomount.sock` | Unix socket path used by CLI and daemon. Use the same config file for mount/status/down when customized. |

On Linux, syslog messages are usually visible with `journalctl -t gomount-daemon -f` when journald collects syslog. On macOS, gomount uses the system syslog compatibility path and does not require Unified Logging integration.

#### SMB (`smb:` block)

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `smb.addr` | Yes | - | SMB server address |
| `smb.port` | No | 445 | SMB server port |
| `smb.share_name` | Yes | - | Share name on the server |
| `smb.username` | Yes | - | Login username |
| `smb.password` | No | - | Login password (prompts if empty) |

#### SSHFS (`sshfs:` block)

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `sshfs.host` | Yes | - | SSH hostname or `~/.ssh/config` alias |
| `sshfs.remote_path` | Yes | - | Remote directory path to mount |

#### WebDAV (`webdav:` block)

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `webdav.url` | Yes | - | WebDAV endpoint URL |
| `webdav.username` | No | - | Login username |
| `webdav.password` | No | - | Login password |
| `webdav.path` | No | - | Path under the WebDAV endpoint |

#### Alibaba Cloud OSS (`aliyun_oss:` block)

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `aliyun_oss.bucket` | Yes | - | OSS bucket name |
| `aliyun_oss.path` | No | - | Prefix to mount within the bucket |
| `aliyun_oss.endpoint` | Yes | - | OSS endpoint, for example `oss-cn-hangzhou.aliyuncs.com` |
| `aliyun_oss.access_key_id` | Yes | - | AccessKey ID |
| `aliyun_oss.access_key_secret` | Yes | - | AccessKey Secret |
| `aliyun_oss.security_token` | No | - | Security Token when using temporary STS credentials |

OSS is object storage and does not provide complete POSIX filesystem semantics. Renames, random writes, and workloads with many small files may be slow; do not use it for databases or busy build directories. Prefer RAM or STS credentials scoped to the target bucket/prefix and protect the configuration file with mode `0600`.

## Usage

### List All Mounts

Display all configured shares with their mount status:

```bash
gomount list
# or
gomount -l
```

### Mount Shares

Mount a specific share by name:

```bash
gomount mount nas1
# or
gomount -m nas1
```

Interactive selection with multi-select support:
- Use `space` to toggle selection on multiple shares
- Press `enter` to confirm and mount all selected shares

```bash
gomount mount
# or
gomount -m
```

**Note**: When mounting multiple shares, if one fails the others will continue. A summary is shown at the end.

### Unmount Shares

Unmount a specific share by name:

```bash
gomount umount nas1
# or
gomount -u nas1
```

Interactive selection with multi-select support:

```bash
gomount umount
# or
gomount -u
```

**Note**: Batch operations continue even if individual operations fail. Success/failure count is displayed at the end.

### Custom Config Path

Use a configuration file from a custom location:

```bash
gomount -c /path/to/config.yaml list
```

### Daemon Management

WebDAV and OSS mounts are managed by the gomount daemon. Check whether it is running:

```bash
gomount daemon status
```

Stop the daemon gracefully. Active WebDAV and OSS sessions are unmounted first; if any unmount fails, the daemon keeps running and reports the failed session.

```bash
gomount daemon down
```

### Help

```bash
gomount --help
gomount list --help
gomount mount --help
gomount umount --help
gomount daemon --help
```

## CLI Reference

```
gomount                          Show help (default)
gomount list                     List all configured mount points
gomount mount [name]             Mount shares (interactive without name)
gomount umount [name]            Unmount shares (interactive without name)
gomount daemon status            Show daemon status and PID
gomount daemon down              Gracefully stop the daemon
gomount config-example           Print example config file

Global Options:
  -c, --config string   Path to config file (default: ~/.config/gomount_config.yaml)
  -h, --help            Show help
```

## TUI Navigation

### List View
- `↑` / `k` - Move cursor up
- `↓` / `j` - Move cursor down
- `g` / `home` - Jump to first item
- `G` / `end` - Jump to last item
- `q` / `esc` - Quit

### Selection Menu (Mount/Unmount)
- `↑` / `k` - Move cursor up
- `↓` / `j` - Move cursor down
- `space` - Toggle selection for current item (multi-select)
- `enter` - Confirm and mount/unmount all selected items
- `q` / `esc` - Cancel

## Security Considerations

- **Password Storage**: Avoid storing passwords in plaintext in the config file. Omit the `password` field to be prompted interactively.
- **File Permissions**: Set restrictive permissions on your config file:
  ```bash
  chmod 600 ~/.config/gomount_config.yaml
  ```
- **Credential Files**: Temporary credential files are created with mode 0600 and deleted immediately after use.

## Development

### Build

```bash
make build
```

### Build for Multiple Platforms

```bash
make build-all
```

### Run Tests

```bash
make test
```

### Lint

```bash
make lint
```

### Format Code

```bash
make fmt
```

## Project Structure

```
gomount/
├── cmd/gomount/            # Application entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── mount/              # Mount/umount operations
│   ├── drivers/            # Protocol drivers (smb, sshfs)
│   ├── tui/                # Terminal UI components
│   ├── interaction/        # Interactive prompts and sudo handling
│   └── privilege/          # Sudo handling
├── configs/                # Example configurations
├── Makefile                # Build automation
└── README.md
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with excellent open-source libraries:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [moby/sys/mountinfo](https://github.com/moby/sys/mountinfo) - Mount status detection
