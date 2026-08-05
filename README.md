# gomount

`gomount` 是一个面向 Linux 和 macOS 的挂载管理工具。

如果你经常需要连接 NAS、远程服务器或云存储，却不想反复查找和输入冗长的挂载命令，可以把连接信息统一写进配置文件，再通过交互式TUI界面选择、挂载和卸载。

支持的挂载种类:
- SMB/CIFS
- SSHFS
- WebDAV
- 阿里云OSS

## 安装gomount

### 系统要求

- Linux下gomount在挂载SMB/CIFS是需要使用到`mount.cifs`命令, 挂载sshfs时需要使用到sshfs命令,因此需要安装相关依赖
- Linux SMB 挂载需要 `sudo` 权限
- macOS 云存储挂载需要 macFUSE、CGO，以及使用 `-tags cmount` 构建的二进制

### 在Linux上安装gomount

**安装依赖**
```bash
# Debian/Ubuntu
sudo apt-get install cifs-utils sshfs

# Fedora/RHEL
sudo dnf install cifs-utils fuse-sshfs

# Arch Linux
sudo pacman -S cifs-utils sshfs
```

**安装gomount**
```bash
go install github.com/hsldymq/gomount/cmd/gomount@latest
```

## macOS上安装gomount

**安装依赖**
```zsh
brew install --cask macfuse
```

**安装gomount**
```zsh
CGO_ENABLED=1 go install -tags cmount github.com/hsldymq/gomount/cmd/gomount@latest
```

## 配置

配置文件用于保存常用的挂载信息。配置一次后，就可以通过交互界面选择挂载项，或直接执行 `gomount mount <名称>`，不必每次重新输入服务器地址、远程路径和挂载位置。

gomount 默认读取 `~/.config/gomount.yaml`。也可以使用 `-c` 指定其他配置文件：

```bash
gomount -c /path/to/config.yaml
```

一个完整的示例配置见[gomount.example.yaml](./examples/gomount.example.yaml)

### 基本结构

配置文件以 `mounts` 开始，其中每一项代表一个可以独立挂载的资源：

```yaml
mounts:
  - name: dev
    type: sshfs
    mount_dir_path: ~/Mounts/dev
    sshfs:
      host: dev.example.com
      remote_path: /home/user/projects
```

每个挂载项只需要关注几个关键部分:

- `name`：挂载项的名称，用于在界面和命令中识别它；名称不能重复。
- `type`：挂载类型，例如 `smb` 或 `sshfs`。
- `mount_dir_path`：资源在本机的挂载位置，支持以 `~` 表示用户目录。
- 与 `type` 同名的配置块：保存连接该资源所需的信息。不同挂载类型需要的字段不同。

### 引用其他配置

当挂载项较多时，可以使用 `include` 把配置拆分到多个文件中：

```yaml
# ~/.config/gomount.yaml
include:
  - mounts/work.yaml
  - mounts/personal/*.yaml

mounts:
  - name: local-nas
    type: smb
    mount_dir_path: ~/Mounts/nas
    smb:
      addr: nas.local
      share_name: data
      username: user
```

被引用的文件仍然使用相同的 `mounts` 结构。`include` 支持单个文件和通配符，也可以继续引用其他配置；相对路径以当前配置文件所在目录为基准，同时支持绝对路径和以 `~` 开头的路径。

拆分配置后，可以按工作、个人、设备或用途分别管理挂载项。这样既能让主配置保持简洁，也便于只同步或分享其中一部分配置；包含密码等敏感信息的文件还可以单独保存并设置更严格的访问权限。
