# gomount

A convenient CLI tool for managing SMB/CIFS mounts on Linux systems with an interactive TUI.

## Features

- **Simple Configuration**: Define all your SMB shares in a single YAML file
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
- `mount.cifs` command (install `cifs-utils` package)
- `sudo` access for mount operations
- Go 1.25+ (for building from source)

### Install Dependencies

On Debian/Ubuntu:
```bash
sudo apt-get install cifs-utils
```

On Fedora/RHEL:
```bash
sudo dnf install cifs-utils
```

On Arch Linux:
```bash
sudo pacman -S cifs-utils
```

## Configuration

Create a configuration file at `~/.config/gomount_config.yaml`:

```yaml
base_dir: /mnt/smb_share

mounts:
  - name: nas1
    type: smb
    smb:
      addr: 10.0.1.2
      port: 445
      share_name: shared_folder
      username: user1
      password: pass1          # Optional, will prompt if omitted
    mount_dir_name: nas1_mount

  - name: media_server
    type: smb
    smb:
      addr: 10.0.1.3
      share_name: media
      username: user2
      # password not stored - will prompt on mount
    mount_dir_path: /mnt/media
```

### Configuration Options

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Unique identifier for this mount |
| `type` | No | auto | Mount type (smb, sshfs, webdav) |
| `smb.addr` | Yes | - | SMB server address |
| `smb.port` | No | 445 | SMB server port |
| `smb.share_name` | Yes | - | Share name on the server |
| `smb.username` | Yes | - | Login username |
| `smb.password` | No | - | Login password (prompts if empty) |
| `mount_dir_name` | No | `<name>` | Directory name within base_dir |
| `mount_dir_path` | No | - | Full custom mount path (overrides base_dir and mount_dir_name) |

An example configuration file is available at `configs/gomount_config.yaml.example`.

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
gomount                  Show help (default)
gomount list             List all configured mount points
gomount mount [name]     Mount SMB shares (interactive without name)
gomount umount [name]    Unmount SMB shares (interactive without name)

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
│   ├── tui/                # Terminal UI components
│   ├── prompt/             # Interactive prompts
│   └── privilege/          # Sudo handling
├── pkg/smb/                # Public types and errors
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