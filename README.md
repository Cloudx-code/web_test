# Music Fetcher - 音乐资源获取器

一个基于 Go 语言的命令行音乐资源获取工具，支持搜索和下载音乐。

## 功能特性

- 关键词搜索歌曲
- 支持多首歌曲批量下载
- 下载进度条实时显示
- 可扩展的 Provider 架构，方便接入更多音乐源
- 当前支持: 酷我音乐

## 项目结构

```
.
├── main.go                          # 程序入口，交互逻辑
├── go.mod                           # Go 模块定义
├── internal/
│   ├── model/
│   │   └── song.go                  # 歌曲数据模型
│   ├── provider/
│   │   ├── provider.go              # Provider 接口定义
│   │   └── kuwo/
│   │       └── kuwo.go              # 酷我音乐实现
│   └── downloader/
│       └── downloader.go            # 文件下载器（带进度条）
└── downloads/                       # 下载文件保存目录（自动创建）
```

## 快速开始

### 环境要求

- Go 1.21+

### 编译运行

```bash
# 编译
go build -o music-fetcher .

# 运行
./music-fetcher
```

或者直接运行：

```bash
go run main.go
```

### 使用方法

1. 启动程序后，输入搜索关键词（歌曲名或歌手名）
2. 从搜索结果中选择要下载的歌曲序号
3. 支持多首下载，序号用逗号分隔（如 `1,3,5`）
4. 下载的文件保存在 `downloads/` 目录中

### 快捷操作

| 按键 | 功能 |
|------|------|
| `q`  | 退出程序 |
| `s`  | 切换音乐来源 |
| `0`  | 返回搜索 |

## 扩展开发

### 添加新的音乐来源

1. 在 `internal/provider/` 下创建新目录
2. 实现 `provider.Provider` 接口：

```go
type Provider interface {
    Name() string
    Search(keyword string, page, pageSize int) ([]model.Song, error)
    GetPlayURL(songID string) (string, error)
}
```

3. 在 `main.go` 中注册新 Provider

## 声明

本项目仅供学习和个人使用，请尊重音乐版权。
