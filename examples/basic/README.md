# 基础配置示例

本目录包含各协议的最小配置示例，适合快速上手。

## 1. SMB 基础配置

文件: `01-smb-basic.yaml`

```yaml
base_dir: /mnt/remote

mounts:
  - name: nas
    smb_addr: 192.168.1.100
    share_name: shared
    username: user
    password: pass
```

使用:
```bash
smb_mount mount nas
```

## 2. SSHFS 基础配置

文件: `02-sshfs-basic.yaml`

```yaml
base_dir: /mnt/remote

mounts:
  - name: dev-server
    type: sshfs
    ssh:
      host: dev.example.com
      user: developer
      key_file: ~/.ssh/id_rsa
    remote_path: /home/developer/projects
```

使用:
```bash
smb_mount mount dev-server
```

## 3. 无密码配置（推荐）

文件: `03-no-password.yaml`

```yaml
base_dir: /mnt/remote

mounts:
  - name: secure-nas
    smb_addr: 192.168.1.100
    share_name: secure
    username: user
    # password 留空，会交互式提示输入
```

使用:
```bash
smb_mount mount secure-nas
# 程序会提示输入密码
```
