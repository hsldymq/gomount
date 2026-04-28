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
    smb:
      addr: 192.168.1.100
      share_name: data
      username: admin
      # password 留空
  # 照片库
  - name: nas-photos
    type: smb
    smb:
      addr: 192.168.1.100
      share_name: photos
      username: admin
  # 视频库
  - name: nas-videos
    type: smb
    smb:
      addr: 192.168.1.100
      share_name: videos
      username: admin
  # 下载机
  - name: nas-downloads
    type: smb
    smb:
      addr: 192.168.1.50
      share_name: downloads
      username: downloader
```
## 方案 B：通过 SSHFS + ProxyJump（无 VPN）
如果你没有 VPN，但家里路由器支持 SSH：
```yaml
mounts:
  # 通过跳板机访问 NAS（在 ~/.ssh/config 中配置 ProxyJump）
  - name: nas-tunnel
    type: sshfs
    sshfs:
      host: home-nas              # ~/.ssh/config 别名（含 ProxyJump）
      remote_path: /data
    mount_dir_path: /mnt/nas-tunnel
  # 直接 SSHFS 到路由器（如果路由器有 USB 存储）
  - name: router-storage
    type: sshfs
    sshfs:
      host: home.ddns.com
      remote_path: /mnt/sda1
    mount_dir_path: /mnt/router-storage
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
