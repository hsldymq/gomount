# gomount

A convenient CLI tool for managing SMB/CIFS, SSHFS, and WebDAV mounts on Linux systems with an interactive TUI.

## Features

- **Simple Configuration**: Define all your mounts in a single YAML file
- **Multiple Protocols**: Supports SMB/CIFS, SSHFS, and WebDAV
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

```bash
go install github.com/hsldymq/gomount/cmd/gomount@latest
```

### Manual Installation

```bash
git clone https://github.com/hsldymq/gomount.git
cd gomount
make build
sudo cp bin/gomount /usr/local/bin/
```

## Requirements

- Linux operating system
- `mount.cifs` command (install `cifs-utils` package) — for SMB mounts
- `sshfs` command — for SSHFS mounts
- `mount.davfs` command (install `davfs2` package) — for WebDAV mounts
- `sudo` access for mount operations
- Go 1.25+ (for building from source)

### Install Dependencies

On Debian/Ubuntu:
```bash
sudo apt-get install cifs-utils sshfs davfs2
```

On Fedora/RHEL:
```bash
sudo dnf install cifs-utils fuse-sshfs davfs2
```

On Arch Linux:
```bash
sudo pacman -S cifs-utils sshfs davfs2
```

## Configuration

Create a configuration file at `~/.config/gomount_config.yaml`:

```yaml
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

  # WebDAV mount
  - name: cloud
    type: webdav
    webdav:
      url: https://cloud.example.com/remote.php/dav/files/user/
      username: user
      # password: pass
    mount_dir_path: /mnt/cloud
```

You can generate a full example config with:

```bash
gomount config-example > ~/.config/gomount_config.yaml
```

### Configuration Options

#### Common Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Unique identifier for this mount |
| `type` | Yes | - | Mount type (`smb`, `sshfs`, `webdav`). |
| `mount_dir_path` | Yes | - | Full local path for the mount point. Supports `~` expansion. |

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
| `webdav.url` | Yes | - | WebDAV server URL |
| `webdav.username` | No | - | Login username |
| `webdav.password` | No | - | Login password (prompts if empty) |

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

### Help

```bash
gomount --help
gomount list --help
gomount mount --help
gomount umount --help
```

## CLI Reference

```
gomount                          Show help (default)
gomount list                     List all configured mount points
gomount mount [name]             Mount shares (interactive without name)
gomount umount [name]            Unmount shares (interactive without name)
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
│   ├── drivers/            # Protocol drivers (smb, sshfs, webdav)
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
