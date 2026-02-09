package handler

import (
	"encoding/json"
	"fmt"
	"music-fetcher/internal/provider/kuwo"
	"net/http"
)

func Play(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "缺少歌曲 ID"})
		return
	}

	p := kuwo.New()
	url, err := p.GetPlayURL(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": fmt.Sprintf("获取链接失败: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "url": url})
}
