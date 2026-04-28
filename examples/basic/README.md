# 基础配置示例

本目录包含各协议的最小配置示例，适合快速上手。

## 1. SMB 基础配置

文件: `01-smb-basic.yaml`

```yaml
base_dir: /mnt/remote

mounts:
  - name: nas
    type: smb
    smb:
      addr: 192.168.1.100
      share_name: shared
      username: user
      password: pass
```

使用:
```bash
gomount mount nas
```

## 2. SSHFS 基础配置

文件: `02-sshfs-basic.yaml`

```yaml
mounts:
  - name: dev-server
    type: sshfs
    sshfs:
      host: dev.example.com
      remote_path: /home/developer/projects
    mount_dir_path: /mnt/dev-projects
```

SSH 连接细节（用户名、端口、密钥、跳板机等）在 `~/.ssh/config` 中配置。

使用:
```bash
gomount mount dev-server
```

## 3. 无密码配置（推荐）

文件: `03-no-password.yaml`

```yaml
base_dir: /mnt/remote

mounts:
  - name: secure-nas
    type: smb
    smb:
      addr: 192.168.1.100
      share_name: secure
      username: user
      # password 留空，会交互式提示输入
```

使用:
```bash
gomount mount secure-nas
# 程序会提示输入密码
```
