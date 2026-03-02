# fs_mount 使用指南

一个支持多种协议（SMB、SSHFS、SSH隧道+SMB、WebDAV）的远程文件系统挂载工具。

## 目录

1. [快速开始](#快速开始)
2. [配置详解](#配置详解)
3. [使用场景](#使用场景)
4. [常见问题](#常见问题)

---

## 快速开始

### 1. 安装依赖

```bash
# Debian/Ubuntu
sudo apt install cifs-utils sshfs davfs2

# Arch Linux
sudo pacman -S cifs-utils sshfs davfs2

# Fedora
sudo dnf install cifs-utils fuse-sshfs davfs2
```

### 2. 创建配置文件

```bash
mkdir -p ~/.config
cp configs/gomount_config.yaml.example ~/.config/gomount_config.yaml
chmod 600 ~/.config/gomount_config.yaml
```

### 3. 编辑配置

```bash
vim ~/.config/gomount_config.yaml
```

### 4. 运行

```bash
# 列出所有挂载
smb_mount list

# 挂载指定条目
smb_mount mount nas

# 卸载指定条目
smb_mount umount nas
```

---

## 配置详解

### 基础配置结构

```yaml

mounts:
  - name: entry1       # 挂载条目名称（唯一标识）
    type: smb          # 驱动类型（可选，自动检测）
    # ... 驱动特定配置

workspaces:
  - name: workspace1   # 工作区名称
    mounts:            # 包含的挂载条目列表
      - entry1
      - entry2
```

### 各协议配置示例

#### SMB/CIFS

```yaml
mounts:
  - name: nas
    type: smb
    smb_addr: 192.168.1.100      # SMB服务器IP
    smb_port: 445                 # 端口（可选，默认445）
    share_name: shared_folder     # 共享名称
    username: user
    password: pass                # 可选，会交互式提示
    mount_dir_name: nas_mount     # 挂载目录名（可选，默认使用name）
```

#### SSHFS

```yaml
mounts:
  - name: dev-server
    type: sshfs
    ssh:
      host: dev.example.com       # SSH服务器
      port: 22                    # SSH端口（可选，默认22）
      user: developer             # 用户名
      key_file: ~/.ssh/id_rsa     # 密钥文件（推荐）
      # password: pass            # 密码（不推荐）
    remote_path: /home/dev/projects  # 远程路径
    options:
      cache_timeout: 600          # 缓存超时（秒）
```

#### SSH隧道 + SMB

```yaml
mounts:
  - name: home-nas-via-ssh
    type: tunnel-smb
    ssh:
      host: home-gateway.com      # SSH跳板机（公网IP）
      user: admin
      key_file: ~/.ssh/home
    smb:
      addr: 192.168.1.100        # 内网SMB服务器IP
      share_name: shared
      username: smbuser
      password: smbpass
```

#### WebDAV

```yaml
mounts:
  - name: nextcloud
    type: webdav
    webdav:
      url: https://cloud.example.com/remote.php/dav/files/user/
      username: user
      password: pass
```

---

## 使用场景

### 场景1：公司开发环境

```yaml
workspaces:
  - name: company-dev
    description: "公司开发环境"
    mounts:
      - nas              # 公司文件服务器
      - dev-server       # 开发服务器
      - gitlab           # GitLab仓库
```

使用方法：
```bash
# 一键挂载整个开发环境
smb_mount workspace company-dev

# 下班一键卸载
smb_mount unworkspace company-dev
```

### 场景2：远程访问家里NAS

不需要VPN，通过SSH隧道直接访问：

```yaml
mounts:
  - name: home-nas
    type: tunnel-smb
    ssh:
      host: your-home-router.com
      user: admin
      key_file: ~/.ssh/home_router
    smb:
      addr: 192.168.1.100
      share_name: photos
      username: nasuser
      password: naspass
```

### 场景3：多云存储管理

```yaml
mounts:
  - name: nextcloud-personal
    type: webdav
    webdav:
      url: https://cloud.personal.com/dav
      username: me
  
  - name: nextcloud-work
    type: webdav
    webdav:
      url: https://cloud.company.com/dav
      username: work

workspaces:
  - name: all-cloud
    mounts:
      - nextcloud-personal
      - nextcloud-work
```

---

## 常见问题

### Q: 如何安全存储密码？

**A:** 推荐方案：
1. 配置文件中不写密码，留空
2. 使用 SSH 密钥认证（SSHFS/SSH隧道）
3. 设置配置文件权限：`chmod 600 ~/.config/gomount_config.yaml`

### Q: SSHFS连接断开怎么办？

**A:** 已内置SSH保活机制，默认每30秒发送心跳包。如需调整：

```yaml
ssh:
  host: example.com
  user: me
  keepalive_interval: 30    # 心跳间隔（秒）
  keepalive_count_max: 3    # 最大失败次数
```

### Q: 如何挂载时不需要sudo？

**A:** 添加用户到fuse组：
```bash
sudo usermod -aG fuse $USER
# 重新登录后生效
```

### Q: 支持Windows吗？

**A:** 目前主要针对Linux/macOS设计。Windows上：
- SMB：使用Windows原生支持
- SSHFS：使用WinFsp + SSHFS-Win
- 本项目暂未在Windows上测试

---

## 更多示例

查看 [examples/](./) 目录下的具体配置文件：

- [basic/](./basic/) - 基础配置示例
- [advanced/](./advanced/) - 高级配置（工作区、SSH隧道等）
- [use-cases/](./use-cases/) - 具体使用场景配置
