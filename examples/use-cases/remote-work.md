# 场景：远程办公配置
适合需要在家访问公司资源的场景。
## 需求
- 访问公司文件服务器（SMB）
- 连接开发服务器（SSHFS）
- 访问内部文档系统（WebDAV）
## 完整配置
```yaml
mounts:
  # 公司文件服务器
  - name: corp-files
    type: smb
    smb_addr: fs.company.local
    share_name: shared
    username: $USER
    # password 留空，挂载时输入
  # 部门专用存储
  - name: dept-storage
    type: smb
    smb_addr: 10.0.10.50
    share_name: engineering
    username: $USER
  # 开发服务器
  - name: dev-box
    type: sshfs
    ssh:
      host: dev.company.com
      user: $USER
      key_file: ~/.ssh/company_id_rsa
    remote_path: /home/$USER/workspace
    options:
      cache_timeout: 600
  # 文档系统（Confluence/SharePoint via WebDAV）
  - name: wiki
    type: webdav
    webdav:
      url: https://wiki.company.com/dav
      username: $USER
workspaces:
  - name: work-mode
    description: "完整办公环境"
    mounts:
      - corp-files
      - dept-storage
      - dev-box
      - wiki
  - name: work-minimal
    description: "轻量级办公（仅必需资源）"
    mounts:
      - corp-files
      - dev-box
```
## 每日工作流
```bash
# 早上开始工作
smb_mount workspace work-mode
# 查看挂载状态
smb_mount list
# ... 工作一整天 ...
# 下班
smb_mount unworkspace work-mode
```
## SSH 隧道版本（适用于无 VPN）
如果公司没有 VPN，但有跳板机：
```yaml
mounts:
  - name: corp-files-via-tunnel
    type: tunnel-smb
    ssh:
      host: jump.company.com      # 跳板机
      user: $USER
      key_file: ~/.ssh/jump_key
    smb:
      addr: fs.company.local      # 内网文件服务器
      share_name: shared
      username: $USER
```
