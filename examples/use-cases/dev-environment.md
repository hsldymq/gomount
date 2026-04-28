# 场景：开发环境配置
适合开发者的日常工作流配置。
## 需求
- 多台开发服务器
- 代码仓库访问
- 构建产物共享
- 测试环境挂载
## 完整配置
```yaml
mounts:
  # 主开发服务器
  - name: dev-primary
    type: sshfs
    sshfs:
      host: dev1.company.com
      remote_path: /home/developer/projects
    options:
      cache_timeout: 600
  # 测试服务器
  - name: dev-test
    type: sshfs
    sshfs:
      host: test.company.com
      remote_path: /var/www/test
  # 生产服务器（只读访问日志）
  - name: prod-logs
    type: sshfs
    sshfs:
      host: prod.company.com
      remote_path: /var/log/app
  # 构建服务器共享
  - name: build-share
    type: smb
    smb:
      addr: build.company.local
      share_name: artifacts
      username: builder
  # 代码审查服务器
  - name: gerrit
    type: webdav
    webdav:
      url: https://gerrit.company.com/dav
      username: developer
workspaces:
  - name: dev-full
    description: "完整开发环境"
    mounts:
      - dev-primary
      - dev-test
      - build-share
      - gerrit
  - name: dev-logs
    description: "仅日志查看"
    mounts:
      - prod-logs
```
## 典型工作流
### 1. 开始一天的工作
```bash
gomount workspace dev-full
# 进入项目目录
cd /mnt/dev/dev-primary/my-project
# 开始编码...
```
### 2. 查看生产日志
```bash
gomount workspace dev-logs
tail -f /mnt/dev/prod-logs/app.log
```
### 3. 部署测试
```bash
# 构建产物在挂载目录中
ls /mnt/dev/build-share/latest/
# 部署到测试环境
scp /mnt/dev/build-share/latest/app.tar.gz dev-test:/var/www/test/
```
### 4. 结束工作
```bash
gomount unworkspace dev-full
```
## VS Code 远程开发配合
可以在 VS Code 中直接打开挂载的目录：
```bash
gomount workspace dev-full
code /mnt/dev/dev-primary/my-project
```
这样就能用本地 VS Code 编辑远程代码。
