package kuwo

import (
	"fmt"
	"handler/internal/model"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	// 旧版搜索 API（无需 token）
	searchAPI = "http://search.kuwo.cn/r.s"
	// 旧版获取下载链接 API
	playURLAPI = "http://antiserver.kuwo.cn/anti.s"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Provider 酷我音乐资源提供者
type Provider struct {
	client *http.Client
}

// New 创建一个新的酷我音乐 Provider
func New() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

func (p *Provider) Name() string {
	return "酷我音乐"
}

func (p *Provider) Search(keyword string, page, pageSize int) ([]model.Song, error) {
	// 旧版 API 页码从 0 开始
	pn := page - 1
	if pn < 0 {
		pn = 0
	}

	params := url.Values{}
	params.Set("all", keyword)
	params.Set("ft", "music")
	params.Set("pn", strconv.Itoa(pn))
	params.Set("rn", strconv.Itoa(pageSize))
	params.Set("rformat", "json")
	params.Set("encoding", "utf8")

	reqURL := searchAPI + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	rawStr := string(body)

	// 旧版 API 返回的是非标准 JSON（单引号），手动解析
	songs := parseSearchResult(rawStr)
	if len(songs) == 0 {
		return nil, nil
	}

	return songs, nil
}

func (p *Provider) GetPlayURL(songID string) (string, error) {
	// songID 格式为 MUSIC_xxxxx 或纯数字
	rid := songID
	if !strings.HasPrefix(rid, "MUSIC_") {
		rid = "MUSIC_" + rid
	}

	params := url.Values{}
	params.Set("type", "convert_url")
	params.Set("rid", rid)
	params.Set("format", "mp3")
	params.Set("response", "url")

	reqURL := playURLAPI + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求播放链接失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	playURL := strings.TrimSpace(string(body))
	if playURL == "" || !strings.HasPrefix(playURL, "http") {
		return "", fmt.Errorf("未获取到有效的播放链接")
	}

	return playURL, nil
}

// parseSearchResult 从旧版 API 的非标准 JSON 中提取歌曲信息
func parseSearchResult(raw string) []model.Song {
	// 找到 abslist 的内容区域
	absIdx := strings.Index(raw, "'abslist':[")
	if absIdx == -1 {
		return nil
	}
	absStart := absIdx + len("'abslist':[")

	// 按 },{  分割每首歌（每首歌是一个顶层对象）
	// 先找到 abslist 数组结尾的 ]
	depth := 1
	absEnd := absStart
	for i := absStart; i < len(raw); i++ {
		switch raw[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				absEnd = i
				goto found
			}
		}
	}
found:
	listStr := raw[absStart:absEnd]

	// 用 "},{" 分隔每首歌，但要在顶层深度分割
	chunks := splitTopLevel(listStr)

	var songs []model.Song
	for _, chunk := range chunks {
		name := extractField(chunk, "SONGNAME")
		if name == "" {
			name = extractField(chunk, "NAME")
		}
		artist := extractField(chunk, "ARTIST")
		album := extractField(chunk, "ALBUM")
		durationStr := extractField(chunk, "DURATION")
		musicRID := extractField(chunk, "MUSICRID")

		if name == "" || musicRID == "" {
			continue
		}

		duration, _ := strconv.Atoi(durationStr)

		songs = append(songs, model.Song{
			ID:       musicRID,
			Name:     name,
			Artist:   artist,
			Album:    album,
			Duration: duration,
			Source:   "kuwo",
		})
	}

	return songs
}

// splitTopLevel 在顶层 {} 深度为 0 的位置按 , 分割
func splitTopLevel(s string) []string {
	var chunks []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				chunk := strings.TrimSpace(s[start:i])
				if chunk != "" {
					chunks = append(chunks, chunk)
				}
				start = i + 1
			}
		}
	}
	if last := strings.TrimSpace(s[start:]); last != "" {
		chunks = append(chunks, last)
	}
	return chunks
}

// extractField 从单引号 JSON 对象字符串中提取指定字段的值
func extractField(item, field string) string {
	// 匹配 'FIELD':'VALUE' 格式
	pattern := regexp.MustCompile(`'` + field + `'\s*:\s*'([^']*)'`)
	matches := pattern.FindStringSubmatch(item)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// SanitizeFileName 清理文件名中的非法字符
func SanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
