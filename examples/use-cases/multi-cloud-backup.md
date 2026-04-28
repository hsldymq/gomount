# 场景：多云备份策略
适合需要将数据备份到多个云存储的场景。
## 需求
- 本地 NAS 作为主存储
- 云端存储作为备份
- 支持多种云服务商
- 一键同步/备份
## 完整配置
```yaml
mounts:
  # 本地主存储
  - name: local-nas
    type: smb
    smb:
      addr: 192.168.1.100
      share_name: data
      username: admin
    mount_dir_path: /mnt/data  # 直接挂载到 /mnt/data
  # Nextcloud（私有云）
  - name: cloud-nextcloud
    type: webdav
    webdav:
      url: https://cloud.personal.com/remote.php/dav/files/me/
      username: me
  # 坚果云（国内）
  - name: cloud-jianguoyun
    type: webdav
    webdav:
      url: https://dav.jianguoyun.com/dav/
      username: email@example.com
  # Dropbox（通过 rclone WebDAV 代理）
  - name: cloud-dropbox
    type: webdav
    webdav:
      url: http://localhost:8001/  # rclone serve webdav 的地址
      username: user
  # OneDrive（通过 rclone WebDAV 代理）
  - name: cloud-onedrive
    type: webdav
    webdav:
      url: http://localhost:8002/
      username: user
```
## 备份工作流
### 1. 挂载所有云存储
```bash
gomount mount cloud-nextcloud
gomount mount cloud-jianguoyun
gomount mount cloud-dropbox
gomount mount cloud-onedrive
```
### 2. 使用 rclone 同步
```bash
# 安装 rclone
# https://rclone.org/
# 同步到 Nextcloud
rclone sync /mnt/data/ remote:nextcloud/backup/ \
  --progress \
  --log-file=/var/log/backup-nextcloud.log
# 同步到坚果云
rclone sync /mnt/data/ remote:jianguoyun/backup/ \
  --progress \
  --log-file=/var/log/backup-jianguoyun.log
```
### 3. 或者使用 rsync
如果已经通过 fs_mount 挂载了：
```bash
# 同步到 Nextcloud
rsync -av --delete \
  /mnt/data/important/ \
  /mnt/backup/cloud-nextcloud/backup/
# 同步到坚果云
rsync -av --delete \
  /mnt/data/documents/ \
  /mnt/backup/cloud-jianguoyun/docs/
```
### 4. 创建自动化脚本
`~/bin/backup-to-cloud.sh`:
```bash
#!/bin/bash
set -e
LOG_FILE="/var/log/cloud-backup-$(date +%Y%m%d).log"
echo "[$(date)] Starting cloud backup..." | tee -a $LOG_FILE
# 挂载所有云存储
for cloud in cloud-nextcloud cloud-jianguoyun cloud-dropbox cloud-onedrive; do
  gomount mount "$cloud" || {
    echo "[$(date)] Failed to mount $cloud" | tee -a $LOG_FILE
    exit 1
  }
done
# 备份到各个云端
for cloud in cloud-nextcloud cloud-jianguoyun cloud-dropbox; do
  echo "[$(date)] Syncing to $cloud..." | tee -a $LOG_FILE
  rsync -av --delete \
    /mnt/data/important/ \
    /mnt/backup/$cloud/backup/ \
    2>&1 | tee -a $LOG_FILE
done
# 卸载
for cloud in cloud-nextcloud cloud-jianguoyun cloud-dropbox cloud-onedrive; do
  echo "[$(date)] Unmounting $cloud..." | tee -a $LOG_FILE
  gomount umount "$cloud"
done
echo "[$(date)] Backup completed!" | tee -a $LOG_FILE
```
### 5. 添加定时任务
```bash
# 编辑 crontab
crontab -e
# 每天凌晨 2 点执行备份
0 2 * * * ~/bin/backup-to-cloud.sh
```
## 安全建议
1. **配置文件权限**：
   ```bash
   chmod 600 ~/.config/fs_mount_config.yaml
   ```
2. **使用应用专用密码**：
   - Nextcloud：生成应用密码
   - 坚果云：使用 WebDAV 专用密码
   - 不要使用主密码
3. **加密敏感数据**：
   ```bash
   # 使用 rclone crypt 加密备份
   rclone sync /mnt/data/secret/ remote:nextcloud/encrypted/ \
     --crypt-remote=nextcloud:encrypted
   ```
## 监控备份状态
```bash
# 查看最近的日志
tail -f /var/log/cloud-backup-*.log
# 检查挂载点
df -h | grep backup
# 验证备份完整性（对比文件数量）
echo "Local: $(find /mnt/data -type f | wc -l) files"
echo "Nextcloud: $(find /mnt/backup/cloud-nextcloud -type f | wc -l) files"
```
