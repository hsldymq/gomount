# gomount

一个方便的 Linux 和 macOS 系统上管理 SMB/CIFS、SSHFS、WebDAV 和阿里云 OSS 挂载的命令行工具，提供交互式 TUI 界面。

## 特性

- **简单配置**：在单个 YAML 文件中定义所有挂载
- **多协议支持**：支持 SMB/CIFS、SSHFS、WebDAV 和阿里云 OSS
- **交互式 TUI**：美观的终端界面用于浏览和选择共享
- **挂载状态跟踪**：查看当前已挂载的共享
- **交互式选择**：轻松选择挂载/卸载操作
- **密码提示**：安全的密码输入，带有视觉反馈
- **权限提升**：需要时自动处理 sudo
- **挂载点管理**：自动创建和清理目录

## 安装

### 从源码安装

```bash
git clone https://github.com/hsldymq/gomount.git
cd gomount
make install
```

### 使用 Go 安装

Linux下执行

```sh
go install github.com/hsldymq/gomount/cmd/gomount@latest
```

MacOS下执行

```sh
CGO_ENABLED=1 go install -tags cmount github.com/hsldymq/gomount/cmd/gomount@latest
```

### 手动安装

```bash
git clone https://github.com/hsldymq/gomount.git
cd gomount
make build
sudo cp bin/gomount /usr/local/bin/
```

## 系统要求

- Linux 或 macOS 操作系统
- Linux：`mount.cifs` 命令（需安装 `cifs-utils` 软件包）— 用于 SMB 挂载
- macOS：`mount_smbfs` 命令 — 用于 SMB 挂载
- `sshfs` 命令 — 用于 SSHFS 挂载
- Linux 云存储挂载需要 FUSE 支持
- macOS 云存储挂载需要 macFUSE、CGO，以及使用 `-tags cmount` 构建的二进制
- Linux SMB 挂载需要 `sudo` 权限
- Go 1.25+ （从源码构建时需要）

在 macOS 上使用 WebDAV 或 OSS 前，安装 macFUSE 并构建启用 macOS 挂载的二进制：

```bash
brew install --cask macfuse
CGO_ENABLED=1 go build -tags cmount -o bin/gomount ./cmd/gomount
```

开发时使用 `CGO_ENABLED=1 go run -tags cmount ./cmd/gomount i`。普通 macOS 构建仍支持 SMB 和 SSHFS，但云存储挂载会返回包含正确构建参数的错误。

### 安装依赖

在 Debian/Ubuntu 上：
```bash
sudo apt-get install cifs-utils sshfs
```

在 Fedora/RHEL 上：
```bash
sudo dnf install cifs-utils fuse-sshfs
```

在 Arch Linux 上：
```bash
sudo pacman -S cifs-utils sshfs
```

## 配置

在 `~/.config/gomount.yaml` 创建配置文件：

```yaml
daemon:
  log_target: syslog                 # syslog、file 或 stderr，默认 syslog
  # log_file: ~/.local/share/gomount/gomount-daemon.log
  # socket_path: ~/.local/share/gomount/gomount.sock

mounts:
  # SMB/CIFS 挂载
  - name: nas
    type: smb
    smb:
      addr: 192.168.1.100
      # port: 445                   # 可选，默认 445
      share_name: shared_folder
      username: user
      # password: pass              # 可选，不填则挂载时交互式输入
    mount_dir_path: /mnt/nas

  # SSHFS 挂载
  # 连接细节（用户名、端口、密钥等）由 ~/.ssh/config 管理
  - name: dev-server
    type: sshfs
    sshfs:
      host: dev.example.com         # ~/.ssh/config 别名或直接 hostname
      remote_path: /home/user/projects
    mount_dir_path: /mnt/dev

  # WebDAV 挂载（由 gomount daemon 管理）
  - name: docs
    type: webdav
    webdav:
      url: https://cloud.example.com/remote.php/dav/files/user/
      username: user                # 可选
      password: pass                # 可选
      path: /team/docs              # WebDAV endpoint 下的可选路径
    mount_dir_path: ~/Mounts/docs

  # 阿里云 OSS 挂载（由 gomount daemon 管理）
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

可以使用以下命令生成完整的配置文件示例：

```bash
gomount config-example > ~/.config/gomount.yaml
```

### 配置选项

#### 通用字段

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `name` | 是 | - | 此挂载的唯一标识符 |
| `type` | 是 | - | 挂载类型（`smb`、`sshfs`、`webdav`、`aliyun_oss`）。 |
| `mount_dir_path` | 是 | - | 挂载点的完整本地路径。支持 `~` 展开。 |

#### Daemon（`daemon:` 块）

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `daemon.log_target` | 否 | `syslog` | daemon 输出目标。`syslog` 不可用时回退 `stderr`；需要固定日志文件时用 `file`，调试时可用 `stderr`。 |
| `daemon.log_file` | 否 | `~/.local/share/gomount/gomount-daemon.log` | `log_target` 为 `file` 时使用的日志文件。 |
| `daemon.socket_path` | 否 | `~/.local/share/gomount/gomount.sock` | CLI 和 daemon 使用的 Unix socket 路径。自定义后 mount/status/down 需要使用同一个配置文件。 |

Linux 上如果 journald 收集 syslog，通常可用 `journalctl -t gomount-daemon -f` 查看。macOS 使用系统 syslog 兼容路径，不依赖 Unified Logging 集成。

#### SMB（`smb:` 块）

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `smb.addr` | 是 | - | SMB 服务器地址 |
| `smb.port` | 否 | 445 | SMB 服务器端口 |
| `smb.share_name` | 是 | - | 服务器上的共享名称 |
| `smb.username` | 是 | - | 登录用户名 |
| `smb.password` | 否 | - | 登录密码（为空时提示输入） |

#### SSHFS（`sshfs:` 块）

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `sshfs.host` | 是 | - | SSH 主机名或 `~/.ssh/config` 别名 |
| `sshfs.remote_path` | 是 | - | 要挂载的远程目录路径 |

#### WebDAV（`webdav:` 块）

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `webdav.url` | 是 | - | WebDAV endpoint URL |
| `webdav.username` | 否 | - | 登录用户名 |
| `webdav.password` | 否 | - | 登录密码 |
| `webdav.path` | 否 | - | WebDAV endpoint 下的路径 |

#### 阿里云 OSS（`aliyun_oss:` 块）

| 字段 | 必需 | 默认值 | 描述 |
|-----|------|--------|------|
| `aliyun_oss.bucket` | 是 | - | OSS Bucket 名称 |
| `aliyun_oss.path` | 否 | - | Bucket 下要挂载的前缀 |
| `aliyun_oss.endpoint` | 是 | - | OSS endpoint，例如 `oss-cn-hangzhou.aliyuncs.com` |
| `aliyun_oss.access_key_id` | 是 | - | AccessKey ID |
| `aliyun_oss.access_key_secret` | 是 | - | AccessKey Secret |
| `aliyun_oss.security_token` | 否 | - | 使用 STS 临时凭据时的 Security Token |

OSS 是对象存储，并不提供完整的 POSIX 文件系统语义。重命名、随机写入和大量小文件操作可能较慢，不建议用作数据库或高频编译目录。建议使用仅授予目标 Bucket/前缀权限的 RAM 或 STS 凭据，并将配置文件权限设为 `0600`。

## 使用方法

### 列出所有挂载点

显示所有配置的共享及其挂载状态：

```bash
gomount list
# 或
gomount -l
```

### 挂载共享

通过名称挂载特定共享：

```bash
gomount mount nas1
# 或
gomount -m nas1
```

交互式选择（支持多选）：
- 使用 `space` 切换多个共享的选中状态
- 按 `enter` 确认并挂载所有选中的共享

```bash
gomount mount
# 或
gomount -m
```

**注意**：挂载多个共享时，如果某个失败，其他共享会继续挂载。最后会显示汇总结果。

### 卸载共享

通过名称卸载特定共享：

```bash
gomount umount nas1
# 或
gomount -u nas1
```

交互式选择（支持多选）：

```bash
gomount umount
# 或
gomount -u
```

**注意**：批量操作时即使单个操作失败也会继续执行。最后会显示成功/失败计数。

### 自定义配置路径

使用来自自定义位置的配置文件：

```bash
gomount -c /path/to/config.yaml list
```

### Daemon 管理

WebDAV 和 OSS 挂载由 gomount daemon 管理。查看 daemon 是否运行：

```bash
gomount daemon status
```

优雅停止 daemon。停止前会先卸载所有 WebDAV session；如果任一卸载失败，daemon 会保持运行并报告失败条目。

```bash
gomount daemon down
```

### 帮助

```bash
gomount --help
gomount list --help
gomount mount --help
gomount umount --help
gomount daemon --help
```

## CLI 参考

```
gomount                          显示帮助（默认）
gomount list                     列出所有配置的挂载点
gomount mount [name]             挂载共享（不带名称时为交互式）
gomount umount [name]            卸载共享（不带名称时为交互式）
gomount daemon status            显示 daemon 状态和 PID
gomount daemon down              优雅停止 daemon
gomount config-example           输出配置文件示例

全局选项：
  -c, --config string   配置文件路径（默认：~/.config/gomount_config.yaml）
  -h, --help            显示帮助
```

## TUI 导航

### 列表视图
- `↑` / `k` - 向上移动光标
- `↓` / `j` - 向下移动光标
- `g` / `home` - 跳到第一项
- `G` / `end` - 跳到最后一项
- `q` / `esc` - 退出

### 选择菜单（挂载/卸载）
- `↑` / `k` - 向上移动光标
- `↓` / `j` - 向下移动光标
- `space` - 切换当前项目的选中状态（多选）
- `enter` - 确认并挂载/卸载所有选中的项目
- `q` / `esc` - 取消

## 安全注意事项

- **密码存储**：避免在配置文件中以明文存储密码。省略 `password` 字段以交互式提示输入。
- **文件权限**：为配置文件设置限制性权限：
  ```bash
  chmod 600 ~/.config/gomount_config.yaml
  ```
- **凭据文件**：临时凭据文件以 0600 权限创建，使用后立即删除。

## 开发

### 构建

```bash
make build
```

### 为多平台构建

```bash
make build-all
```

### 运行测试

```bash
make test
```

### 代码检查

```bash
make lint
```

### 格式化代码

```bash
make fmt
```

## 项目结构

```
gomount/
├── cmd/gomount/            # 应用入口点
├── internal/
│   ├── config/             # 配置管理
│   ├── mount/              # 挂载/卸载操作
│   ├── drivers/            # 协议驱动（smb、sshfs）
│   ├── tui/                # 终端 UI 组件
│   ├── interaction/        # 交互式提示和 Sudo 处理
│   └── privilege/          # Sudo 处理
├── configs/                # 示例配置
├── Makefile                # 构建自动化
└── README.md
```

## 贡献

欢迎贡献！请随时提交 Pull Request。

## 许可证

本项目在 MIT 许可证下授权 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

使用优秀的开源库构建：
- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - 样式
- [moby/sys/mountinfo](https://github.com/moby/sys/mountinfo) - 挂载状态检测
