package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	downloadDir = "downloads"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// progressWriter 包装 io.Writer 以显示下载进度
type progressWriter struct {
	total       int64
	written     int64
	barWidth    int
	lastPercent int
}

func newProgressWriter(total int64) *progressWriter {
	return &progressWriter{
		total:       total,
		barWidth:    40,
		lastPercent: -1,
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	pw.printProgress()
	return n, nil
}

func (pw *progressWriter) printProgress() {
	if pw.total <= 0 {
		// 未知大小，只显示已下载量
		fmt.Fprintf(os.Stderr, "\r  下载中... %s", formatBytes(pw.written))
		return
	}

	percent := int(float64(pw.written) / float64(pw.total) * 100)
	if percent == pw.lastPercent {
		return
	}
	pw.lastPercent = percent

	filled := pw.barWidth * percent / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pw.barWidth-filled)

	fmt.Fprintf(os.Stderr, "\r  下载中 [%s] %3d%% %s/%s",
		bar, percent,
		formatBytes(pw.written),
		formatBytes(pw.total))
}

func (pw *progressWriter) finish() {
	fmt.Fprintln(os.Stderr)
}

// formatBytes 将字节数格式化为可读字符串
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// Download 下载文件到 downloads 目录
// url: 下载链接
// filename: 保存的文件名
func Download(downloadURL, filename string) (string, error) {
	// 确保下载目录存在
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("创建下载目录失败: %w", err)
	}

	savePath := filepath.Join(downloadDir, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(savePath); err == nil {
		return savePath, fmt.Errorf("文件已存在: %s", savePath)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，HTTP 状态码: %d", resp.StatusCode)
	}

	// 创建文件
	file, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 带进度条下载
	pw := newProgressWriter(resp.ContentLength)
	_, err = io.Copy(file, io.TeeReader(resp.Body, pw))
	pw.finish()

	if err != nil {
		// 下载失败，删除不完整的文件
		os.Remove(savePath)
		return "", fmt.Errorf("下载文件失败: %w", err)
	}

	return savePath, nil
}
