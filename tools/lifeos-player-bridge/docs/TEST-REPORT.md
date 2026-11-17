# Player Bridge 测试报告

记录时间：2026-11-17 · 版本：`player-bridge 0.1.0`

## 结论摘要

| 阶段 | 内容 | 状态 | 证据 |
| --- | --- | --- | --- |
| P1 | 配置读取、PotPlayer 检测、凭据读取、Jellyfin PlaybackInfo、容器路径解析 | ✅ 已通过（真实机器） | `-verify` 5 项全 OK：config / credential / jellyfin(10.10.7) / player(PotPlayerMini64.exe) / mapping |
| P2 | 容器路径 → 盘符映射、文件存在性、错误路径拒绝 | ✅ 已通过（真实机器 + 单元测试） | `-resolve` 输出 `src /media/电影/…mkv (size=10330139819)` → `OK path V:\…\media\电影\…mkv (10330139819 bytes)`；`pathmap` 包测试覆盖前缀边界、`..`、扩展名、盘符不可用 |
| P3 | PotPlayer 真实启动播放 NAS 文件 | ✅ 已通过（真实机器） | 用户机器执行 `-play <itemId>` 后 PotPlayer 正常打开并播放；随后从 LifeOS 页面点按钮再次验证通过 |
| P4 | LifeOS 电影详情播放入口 → 浏览器 → 协议 → 播放 | ✅ 已通过（真实浏览器） | 云端页面点「使用 PotPlayer 播放」→ 浏览器弹出「此站点正在尝试打开 player-bridge.exe / 43.138.202.56:8888 想打开此应用程序」→ PotPlayer 打开并播放 `Love.Letter.1995.REPACK.1080p…mkv`（进度 01:36:23）；页面状态文案显示「已向 Windows 提交启动请求…播放是否成功请以 PotPlayer 是否打开为准」 |
| P5 | 稳定性用例 | ⏳ 部分 | 见下表 |

## P5 稳定性用例

| 用例 | 方法 | 结果 |
| --- | --- | --- |
| Jellyfin 不可达 | 指向不可达地址 | ✅ 退出码 5，提示检查家庭网络与 NAS |
| Jellyfin 响应慢 | 桩服务延迟 1.5s×2 + `timeout_seconds: 1` | ✅ 退出码 5，提示 `did not answer in time`，并建议调大 `timeout_seconds` |
| 映射盘不可用 | 目标盘根不存在 | ✅ `ErrDriveUnavailable`，提示检查 SMB |
| 文件被删除 | `-resolve` 指向不存在文件 | ✅ 退出码 8，`media file is missing on the mapped drive` |
| 不支持的格式 | `-probe /media/movies/setup.exe` | ✅ 拒绝：`extension is not allowed: .exe` |
| 非法 ItemId | `-play ../etc/passwd` | ✅ 退出码 2，`invalid item id` |
| 路径穿越 | `/media/movies/../../Windows/notepad.exe` | ✅ 拒绝（`..` 或未命中映射） |
| 中文文件名 | `-probe /media/电影/Your.Name…mkv` | ✅ 正确映射 |
| 含空格文件名 | `-probe /media/电影/中文 名称 with space.mp4` | ✅ 正确映射 |
| PotPlayer 不存在 | 开发机（无 PotPlayer） | ✅ 退出码 9，`PotPlayer not found` |
| API 密钥访问 `/Users/Me` | Jellyfin 返回 400 | ✅ `setup.ps1` 自动退到 `/Users`，或指定 `-JellyfinUser` |
| 凭据 UTF-16 | `cmdkey` 写入 | ✅ 修复后 `DecodeBlob` 正确解码（此前导致 `invalid header field value`） |
| 协议唤起时闪控制台窗口 | 以 `-H=windowsgui` 构建 + `AttachConsole(ATTACH_PARENT_PROCESS)` | ✅ 协议启动不再分配控制台；终端里 `-verify`/`-play` 仍能打印（PE subsystem=GUI 实测） |

## 单元测试

```
go vet ./...            干净
go test ./...           7 个包全部 ok
  protocol   URL 白名单 + 10 组非法输入拒绝
  config     规范化与危险配置拒绝（缺映射、ftp、相对路径、POSIX target、错误播放器）
  pathmap    前缀边界、穿越拒绝、扩展名、文件缺失/盘符不可用
  jellyfin   状态码→具名错误、PlaybackInfo 解析、库白名单（含空白名单拒绝）
  security   UTF-16LE / UTF-8 凭据解码
  logger     显式密钥与 token 字段脱敏，item id 保持可读
  player     显式 exe 检测、缺失拒绝
```

## 真实机器证据（目标 Windows 主机）

```
> .\dist\player-bridge.exe -verify
OK    config    C:\Users\Crazy\AppData\Local\LifeOS\PlayerBridge\config.yaml
OK    credential lifeos-player-bridge/jellyfin
OK    jellyfin  http://192.168.31.122:8096            (10.10.7)
OK    player    C:\Program Files\DAUM\PotPlayer\PotPlayerMini64.exe
OK    mapping   V:\satall-15071680779a\jellyfin\media\电影\

> .\dist\player-bridge.exe -resolve 8d2ea84ddeb6bbcaca147219271e3cde88
libs  电影[movies] > Media Folders[]
OK    library
src   /media/电影/Your.Name…mkv (container=mkv size=10330139819)
OK    path  V:\…\media\电影\Your.Name…\Your.Name…mkv (10330139819 bytes)
OK    player C:\Program Files\DAUM\PotPlayer\PotPlayerMini64.exe
```

## 已知限制

- 浏览器到自定义协议的结果没有回调：UI 只显示"已提交启动请求"，不代表播放成功；真实结果以 PotPlayer 是否打开为准。
- `allowed_libraries` 默认 `movies`（Jellyfin 的 CollectionType），与目录名无关；库名不同需自行补。
- 电视剧单集播放未实现（按任务书属后续扩展），Bridge 本身按单 itemId 工作。