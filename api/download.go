package handler

import (
	"encoding/json"
	"fmt"
	"handler/internal/provider/kuwo"
	"io"
	"net/http"
	"strings"
)

func Download(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("name")
	isPlay := r.URL.Query().Get("play") == "1"
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "缺少歌曲 ID"})
		return
	}
	if name == "" {
		name = id
	}

	p := kuwo.New()
	playURL, err := p.GetPlayURL(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": fmt.Sprintf("获取链接失败: %v", err)})
		return
	}

	req, err := http.NewRequest(http.MethodGet, playURL, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "创建请求失败"})
		return
	}
	if userAgent := r.UserAgent(); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "下载失败"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		if isPlay {
			http.Error(w, "播放失败", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "下载源返回异常"})
		return
	}

	for _, key := range []string{
		"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges",
	} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}

	safeName := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\"", "_").Replace(name)
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	if isPlay {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp3"`, safeName))
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
