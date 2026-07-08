# 场景：远程办公配置
适合需要在家访问公司资源的场景。
## 需求
- 访问公司文件服务器（SMB）
- 连接开发服务器（SSHFS）
## 完整配置
```yaml
mounts:
  # 公司文件服务器
  - name: corp-files
    type: smb
    smb:
      addr: fs.company.local
      share_name: shared
      username: $USER
      # password 留空，挂载时输入
  # 部门专用存储
  - name: dept-storage
    type: smb
    smb:
      addr: 10.0.10.50
      share_name: engineering
      username: $USER
  # 开发服务器
  - name: dev-box
    type: sshfs
    sshfs:
      host: dev.company.com
      remote_path: /home/$USER/workspace
    options:
      cache_timeout: 600
```
## 每日工作流
```bash
# 早上开始工作
gomount mount corp-files
gomount mount dept-storage
gomount mount dev-box
# 查看挂载状态
gomount list
# ... 工作一整天 ...
# 下班
gomount umount corp-files
gomount umount dept-storage
gomount umount dev-box
```
## 通过跳板机访问（适用于无 VPN）
如果公司没有 VPN，但有跳板机，在 `~/.ssh/config` 中配置 ProxyJump 即可：
```yaml
mounts:
  - name: corp-files-via-tunnel
    type: sshfs
    sshfs:
      host: corp-files             # ~/.ssh/config 别名（含 ProxyJump jump.company.com）
      remote_path: /data/shared
    mount_dir_path: /mnt/corp-files
```
