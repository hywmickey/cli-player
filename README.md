# cli-player

一个用 Golang 编写的命令行音乐播放器，支持歌词文件显示和可视化界面。

## 功能特性

- 命令行终端用户界面（TUI）
- 支持多种音频格式（MP3、FLAC、OGG 等）
- 歌词文件显示
- 播放列表管理

## 环境要求

- Go 1.25.0 或更高版本
- 操作系统音频驱动支持

## 编译方法

```bash
# 克隆项目
git clone <repository-url>
cd cli-player

# 下载依赖
go mod download

# 编译
go build -o cli-player .
```

## 运行方法

```bash
# 指定音频文件目录运行
./cli-player /path/to/audio/files

# 指定单个音频文件运行
./cli-player /path/to/audio.mp3

# 当前目录运行
./cli-player .
```

## 快捷键

| 按键 | 功能 |
|------|------|
| `↑/↓` | 导航列表 |
| `Enter` | 播放选中歌曲 |
| `Space` | 暂停/继续 |
| `q` | 退出程序 |

## 依赖

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [beep](https://github.com/faiface/beep) - 音频播放库
- [lipgloss](https://github.com/charmbracelet/lipgloss) - 样式库
