# 场景：家庭 NAS 配置
适合拥有家庭 NAS 的用户，需要多设备访问家庭存储。
## 需求
- 在外地访问家里 NAS 文件
- 照片/视频/文档的统一存储
- 下载机共享访问
## 方案 A：通过 VPN（推荐）
如果你已经配置了 WireGuard/OpenVPN：
```yaml
mounts:
  # 主 NAS
  - name: nas-main
    type: smb
    smb_addr: 192.168.1.100
    share_name: data
    username: admin
    # password 留空
  # 照片库
  - name: nas-photos
    type: smb
    smb_addr: 192.168.1.100
    share_name: photos
    username: admin
  # 视频库
  - name: nas-videos
    type: smb
    smb_addr: 192.168.1.100
    share_name: videos
    username: admin
  # 下载机
  - name: nas-downloads
    type: smb
    smb_addr: 192.168.1.50
    share_name: downloads
    username: downloader
workspaces:
  - name: all-home
    description: "所有家庭存储"
    mounts:
      - nas-main
      - nas-photos
      - nas-videos
      - nas-downloads
```
## 方案 B：通过 SSH 隧道（无 VPN）
如果你没有 VPN，但家里路由器支持 SSH：
```yaml
mounts:
  # 通过 SSH 隧道访问 NAS
  - name: nas-tunnel
    type: tunnel-smb
    ssh:
      host: home.ddns.com         # 家里路由器的 DDNS 地址
      port: 22
      user: admin
      key_file: ~/.ssh/home_router
    smb:
      addr: 192.168.1.100        # NAS 内网 IP
      share_name: data
      username: nasuser
  # 直接 SSHFS 到路由器（如果路由器有 USB 存储）
  - name: router-storage
    type: sshfs
    ssh:
      host: home.ddns.com
      user: admin
      key_file: ~/.ssh/home_router
    remote_path: /mnt/sda1
workspaces:
  - name: home-remote
    description: "远程访问家里存储"
    mounts:
      - nas-tunnel
      - router-storage
```
## 照片备份工作流
```bash
# 挂载照片库
gomount mount nas-photos
# 同步照片
rsync -av ~/Pictures/ /mnt/home/nas-photos/backup/
# 卸载
gomount umount nas-photos
```
