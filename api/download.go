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

	resp, err := http.Get(playURL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "下载失败"})
		return
	}
	defer resp.Body.Close()

	safeName := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\"", "_").Replace(name)
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp3"`, safeName))
	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}
	io.Copy(w, resp.Body)
}
