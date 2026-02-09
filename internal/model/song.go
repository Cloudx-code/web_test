package model

import "fmt"

// Song 表示一首歌曲的基本信息
type Song struct {
	ID       string // 歌曲唯一标识
	Name     string // 歌曲名称
	Artist   string // 歌手
	Album    string // 专辑
	Duration int    // 时长（秒）
	Source   string // 来源平台
}

// FormatDuration 将秒数格式化为 mm:ss
func (s Song) FormatDuration() string {
	min := s.Duration / 60
	sec := s.Duration % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}
