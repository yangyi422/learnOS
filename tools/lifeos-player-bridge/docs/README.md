# LifeOS Player Bridge（Windows）

把 LifeOS 媒体中心里的电影交给本机 PotPlayer 播放：浏览器只发出一个自定义协议，
Bridge 用专用 Jellyfin 用户凭证换取 PlaybackInfo，把容器路径映射成本机盘符路径，
校验文件真实存在后启动 PotPlayer。视频不经过 LifeOS 服务器，也不经过 Mac mini。

## 运行模型

```
LifeOS（Vue）--lifeos-player://play?itemId=<JellyfinItemId>--> player-bridge.exe
                                                                  |  Jellyfin PlaybackInfo
                                                                  |  + 媒体库白名单校验
                                                                  v
                                                    路径映射（容器路径 → 盘符）
                                                                  v
                                                     PotPlayer <file on NAS SMB>
```

- 只在播放时启动，启动完立即退出；**不监听端口、没有常驻服务**。
- 参数来自 URL，但 URL 只允许 `play?itemId=<32 位 Jellyfin ID>`，其余一律拒绝。
- 不通过 cmd.exe 拼接，使用 `exec.Command(player, mediaPath)` 单独传参。

## 安装

```powershell
cd tools\lifeos-player-bridge
.\scripts\build.ps1                       # 生成 dist\player-bridge.exe（x64）
.\scripts\install.ps1                     # 注册 HKCU 协议 + 创建配置目录 + 自检
# 可选：把 Jellyfin token 存进 Windows 凭据管理器（不会写进配置文件）
.\scripts\install.ps1 -JellyfinToken '<token>' -JellyfinUser '<jellyfin-user-id>'
```

安装脚本做的事：检测架构 → 检测 PotPlayer → 复制配置模板（不覆盖已有配置）→ 写
`HKCU\Software\Classes\lifeos-player` → 回读校验 → 运行 `player-bridge.exe -verify` → 打印当前映射盘。
不需要管理员权限。

```powershell
.\scripts\uninstall.ps1                    # 只删 HKCU 协议注册
```
卸载不会删除配置、不删除凭据、不影响 PotPlayer 自身注册。凭据需要手动清理：
`cmdkey /delete:lifeos-player-bridge/jellyfin`。

## 配置

安装后编辑 `%LOCALAPPDATA%\LifeOS\PlayerBridge\config.yaml`（模板见 `config.example.yaml`）：

| 字段 | 说明 |
| --- | --- |
| `jellyfin.base_url` | Jellyfin 地址，例如 `http://192.168.31.122:8096` |
| `jellyfin.user_id` | **专用非管理员** Jellyfin 用户的 ID（Dashboard → 用户 → 该用户的 ID） |
| `jellyfin.credential_target` | 凭据管理器条目名，默认 `lifeos-player-bridge/jellyfin` |
| `jellyfin.allowed_libraries` | 允许播放的媒体库：CollectionType（如 `movies`）或库名（如 `电影`） |
| `player.executable` | `auto` 自动搜索 PotPlayer，或写绝对路径 |
| `path_mappings[]` | 容器内路径前缀 → 本机盘符前缀，**必须**以 `/` 和 `\` 结尾 |
| `playback.check_file_exists` | 启动前检查文件存在（建议 `true`） |
| `playback.allowed_extensions` | 允许的容器扩展名 |

### 路径映射怎么确认

1. 在 NAS 上查看 Jellyfin 容器的挂载：`docker inspect jellyfin --format '{{json .Mounts}}'`，
   记下宿主机目录与容器内目录（例如 `/volume1/media/movies` → `/media/movies`）。
2. 在 Windows 上确认 SMB 映射：`net use` 或 `Get-PSDrive -PSProvider FileSystem`，
   记下盘符对应的共享（例如 `Z:` → `\\192.168.31.122\media`）。
3. 把**容器内**前缀写进 `source`，把**盘符**前缀写进 `target`。

Bridge 不做任何猜测：前缀不匹配、路径含 `..`、扩展名不在白名单、映射盘不可达、
文件不存在，都会以明确错误退出，不会退化成“随便打开一个路径”。

## 凭证

两种 token，权限差别很大，先选对：

| 方式 | 怎么拿 | 权限 | 推荐 |
| --- | --- | --- | --- |
| 专用用户 token | `POST /Users/AuthenticateByName`（用户名+密码），或 Jellyfin 里该用户登录后的 `AccessToken` | 受该用户的媒体库权限约束 | ✅ 推荐 |
| API 密钥 | 控制台 → API 密钥 | 服务端级，等于全库可见；此时唯一限制是 `allowed_libraries` | 仅在无法建用户时使用 |

`/Users/Me` 对 **API 密钥** 会返回 400（API 密钥不绑定用户），这不是 token 写错；`setup.ps1` 会自动退到 `/Users` 再让人选，或直接用 `-JellyfinUser` 指定。

```powershell
cmdkey /generic:lifeos-player-bridge/jellyfin /user:<jellyfin-user-id> /pass:<api-token-or-user-token>
```

- 只放在 Windows 凭据管理器（CRED_TYPE_GENERIC），**不写配置文件、不进 URL、不进日志**。
- 使用专用非管理员用户，且该用户只能看到允许播放的媒体库；Bridge 还会再用
  `allowed_libraries` 校验一次 Item 的祖先库，所以改 URL 里的 itemId 也拿不到未授权媒体。

### 用 -probe 验证配对是否正确

`-verify` 只检查目标根目录存在，**不证明路径配对正确**。配对是否成立，用一条真实路径试：

```powershell
# 容器路径从 Jellyfin 某个条目的 PlaybackInfo 里取
.\dist\player-bridge.exe -probe '/media/movies/Dune.mkv'
```

```
rule  /media/movies/ -> Q:\Movies\
OK    Q:\Movies\Dune.mkv (26843545600 bytes)
```

退化为 `media file is missing on the mapped drive`（退出码 8）说明前缀对了但文件不在，通常是
target 少一层目录；`no path mapping matched`（退出码 7）说明 source 前缀与 Jellyfin 返回的容器路径不一致。

## 自检与排错

```powershell
cd tools\lifeos-player-bridge\dist
.\player-bridge.exe -verify      # 配置 / 凭据 / Jellyfin / PotPlayer / 映射盘逐项检查
.\player-bridge.exe 'lifeos-player://play?itemId=<32 位 itemId>'
```

日志：`%LOCALAPPDATA%\LifeOS\PlayerBridge\logs\bridge.log`（token 与长串密钥会被替换为 `***`）。

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 退出码 2 / 日志 `protocol` | URL 形状不对或 itemId 非法 | 从 LifeOS 详情页重新点播放 |
| 退出码 3 | 配置缺失或字段非法 | 按 `config.example.yaml` 补全 |
| 退出码 4 | 凭据不存在 | 用上面的 `cmdkey` 写入 |
| 退出码 5 | Jellyfin 不可达 / token 被拒 / item 不存在 | 检查家庭网络、专用用户权限 |
| 退出码 6 | item 不在允许的媒体库 | 检查 `allowed_libraries` 与 Jellyfin 权限 |
| 退出码 7 | 路径未命中映射 | 校正 `path_mappings` |
| 退出码 8 | 文件或盘符不可用 | 重新连接 SMB 映射盘 |
| 退出码 9 | PotPlayer 未安装或启动失败 | 安装 PotPlayer 或指定 `player.executable` |

按 item id 一次性诊断（库白名单 → 容器路径 → Windows 路径 → 文件 → 播放器），**不启动播放器**：

```powershell
.\dist\player-bridge.exe -resolve 0f8fad5bd9cb469fa16570867728950e
```

```
libs  Movies[movies] > NAS[]
OK    library
src   /media/movies/Dune.mkv (container=mkv size=1024)
OK    path D:\tmp-movies\Dune.mkv (1 bytes)
FAIL  player — PotPlayer not found
```

退出码与真实播放一致（6 库、7 映射、8 文件、9 播放器），所以这一条命令就能回答"为什么点播放没反应"。

配对调试用 `player-bridge.exe -probe '<容器路径>'`：打印命中的规则、映射后的 Windows 路径与文件大小，
不需要启动播放器。

## 测试

```powershell
$env:GOPATH='..\..\.tmp\gopath'; go test ./...
```

覆盖：协议白名单与非法参数拒绝、配置校验与路径规范、路径映射边界与穿越拒绝、
扩展名白名单、Jellyfin 状态码到错误类型的映射、媒体库白名单、日志脱敏、PotPlayer 检测。
凭据读取依赖 Windows API，由 `-verify` 与真实播放验证，不写单元测试。

## 未包含（延期）

- 电视剧单集播放（Bridge 只按单个 itemId 工作，扩展时从 Jellyfin 取 Season/Episode 即可）。
- 安装包签名、自动更新、多播放器支持。