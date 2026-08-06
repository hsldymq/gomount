# gomount
<!-- TOC -->

- [gomount](#gomount)
    - [安装gomount](#%E5%AE%89%E8%A3%85gomount)
        - [系统要求](#%E7%B3%BB%E7%BB%9F%E8%A6%81%E6%B1%82)
        - [在Linux上安装gomount](#%E5%9C%A8linux%E4%B8%8A%E5%AE%89%E8%A3%85gomount)
        - [在macOS上安装gomount](#%E5%9C%A8macos%E4%B8%8A%E5%AE%89%E8%A3%85gomount)
    - [配置](#%E9%85%8D%E7%BD%AE)
        - [基本结构](#%E5%9F%BA%E6%9C%AC%E7%BB%93%E6%9E%84)
        - [引用其他配置](#%E5%BC%95%E7%94%A8%E5%85%B6%E4%BB%96%E9%85%8D%E7%BD%AE)
    - [使用方法](#%E4%BD%BF%E7%94%A8%E6%96%B9%E6%B3%95)
        - [交互式挂载](#%E4%BA%A4%E4%BA%92%E5%BC%8F%E6%8C%82%E8%BD%BD)
        - [挂载指定项目](#%E6%8C%82%E8%BD%BD%E6%8C%87%E5%AE%9A%E9%A1%B9%E7%9B%AE)
        - [卸载指定项目](#%E5%8D%B8%E8%BD%BD%E6%8C%87%E5%AE%9A%E9%A1%B9%E7%9B%AE)
        - [查看挂载列表](#%E6%9F%A5%E7%9C%8B%E6%8C%82%E8%BD%BD%E5%88%97%E8%A1%A8)
        - [后台 daemon](#%E5%90%8E%E5%8F%B0-daemon)
            - [什么时候会启动](#%E4%BB%80%E4%B9%88%E6%97%B6%E5%80%99%E4%BC%9A%E5%90%AF%E5%8A%A8)
            - [查看运行状态](#%E6%9F%A5%E7%9C%8B%E8%BF%90%E8%A1%8C%E7%8A%B6%E6%80%81)
            - [什么时候需要关闭](#%E4%BB%80%E4%B9%88%E6%97%B6%E5%80%99%E9%9C%80%E8%A6%81%E5%85%B3%E9%97%AD)

<!-- /TOC -->

`gomount` 是一个面向 Linux 和 macOS 的挂载管理工具。

如果你经常需要连接 NAS、远程服务器或云存储，却不想反复查找和输入冗长的挂载命令，可以把连接信息统一写进配置文件，再通过交互式TUI界面选择、挂载和卸载。

支持的挂载种类:
- SMB/CIFS
- SSHFS
- WebDAV
- S3 及兼容存储（AWS S3、阿里云 OSS、腾讯云 COS、七牛云 Kodo、MinIO、RustFS 等）

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

### 在macOS上安装gomount

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

## 使用方法

### 交互式挂载

交互式界面是 gomount 最主要的使用方式。运行以下命令进入界面：

```bash
gomount i
```

`i` 是 `interactive` 的简写，也可以使用完整命令：

```bash
gomount interactive
```

进入界面后，gomount 会读取配置并检查每个挂载项的当前状态。列表中会显示挂载项的名称、远程地址和类型：已挂载的项目显示为选中状态，未挂载的项目显示为未选中状态。

界面大致如下，实际显示时会使用颜色区分不同状态：

```text
Select shares to mount/unmount:

▸ [✓] home-nas › //192.0.2.10:445/documents (smb)
  [ ] dev-server › dev-server:/srv/projects (sshfs)
  [ ] s3-archive › s3://example-bucket/backups@oss-cn-hangzhou.aliyuncs.com (s3)
  [✓] team-docs › https://dav.example.com/dav/:/team/docs (webdav)

↑/k: up | ↓/j: down | space: select | enter: confirm | q/esc: cancel
```

- 使用 `↑`、`↓` 或 `k`、`j` 移动光标。
- 按 `Space` 切换当前项目的目标状态，可以一次选择多个项目。
- 按 `Enter` 确认，gomount 会挂载新选中的项目，并卸载取消选中的项目。
- 按 `q`、`Esc` 或 `Ctrl+C` 退出，不执行任何操作。

如果使用的不是默认配置文件，可以在进入交互界面时通过 `-c` 指定：

```bash
gomount -c /path/to/config.yaml i
```

### 挂载指定项目

不进入交互界面，直接按配置中的名称挂载一个或多个项目：

```bash
gomount m home-nas
gomount m home-nas dev-server
```

`m` 是 `mount` 的简写，完整命令为 `gomount mount <名称>`。

### 卸载指定项目

按配置中的名称卸载一个或多个项目：

```bash
gomount u home-nas
gomount u home-nas dev-server
```

`u` 是 `unmount` 的简写，完整命令为 `gomount unmount <名称>`。

### 查看挂载列表

查看所有配置项及其当前挂载状态：

```bash
gomount l
```

`l` 是 `list` 的简写，完整命令为 `gomount list`。

### 后台 daemon

除了直接执行命令，gomount 还可能在后台启动一个 daemon 进程。

SMB 和 SSHFS 的挂载由系统中的 `mount.cifs`、`mount_smbfs` 或 `sshfs` 等工具完成，gomount 发出挂载命令后即可退出。其他挂载类型则由 gomount 内部的挂载引擎提供服务，需要有一个持续运行的进程来维持挂载、处理文件读写并记录挂载状态，因此由 daemon 统一管理。

#### 什么时候会启动

当你通过交互界面或 `gomount m` 挂载一个需要 daemon 管理的项目时，gomount 会先检查 daemon 是否已经运行：

- 如果已经运行，CLI 会直接把挂载请求交给它。
- 如果尚未运行，CLI 会自动在后台启动它，等待就绪后再继续挂载。
- 如果只使用 SMB 和 SSHFS，gomount 不会因此启动 daemon。

daemon 启动后独立于当前这次 CLI 操作运行。关闭交互界面或退出当前终端命令，不会同时关闭 daemon；即使它管理的挂载项已经全部卸载，daemon 也不会自动退出。之后再次挂载时，gomount 会继续使用现有进程。

通常不需要手动启动 daemon，也不需要在每次使用完 gomount 后立即关闭它。

#### 查看运行状态

可以查看 daemon 是否正在运行、进程 ID，以及当前由它维持的挂载数量：

```bash
gomount d status
```

`d` 是 `daemon` 的简写，完整命令为 `gomount daemon status`。

#### 什么时候需要关闭

以下情况适合手动关闭 daemon：

- 暂时不再使用由 daemon 管理的挂载，希望释放后台进程占用的资源。
- 准备退出系统、维护挂载环境，或希望先干净地卸载相关资源。
- 升级了 gomount，希望结束旧版本的 daemon，让下次挂载时自动启动新版本。
- daemon 状态异常，需要在确认不再使用相关挂载后重新启动。

使用以下命令关闭：

```bash
gomount d down
```

关闭时，gomount 会先卸载该 daemon 当前管理的所有挂载，再停止后台进程。这意味着正在使用这些挂载目录的程序会失去访问，因此不要在仍有文件读写、终端位于挂载目录中，或其他程序正在使用相关文件时执行此命令。

如果某个挂载无法安全卸载，命令会报告错误，daemon 也会继续运行，以免在挂载尚未清理完成时直接退出。相比直接使用 `kill` 终止进程，应优先使用 `gomount d down`。

如果配置中自定义了 daemon 的通信路径，查询状态和关闭时需要通过 `-c` 使用同一份配置：

```bash
gomount -c /path/to/config.yaml d status
gomount -c /path/to/config.yaml d down
```
