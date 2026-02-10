package provider

import "handler/internal/model"

// Provider 定义音乐资源提供者的接口
type Provider interface {
	// Name 返回提供者名称
	Name() string

	// Search 根据关键词搜索歌曲
	// keyword: 搜索关键词
	// page: 页码，从 1 开始
	// pageSize: 每页数量
	Search(keyword string, page, pageSize int) ([]model.Song, error)

	// GetPlayURL 获取歌曲播放/下载链接
	// songID: 歌曲 ID
	GetPlayURL(songID string) (string, error)
}
