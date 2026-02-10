package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func Play(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "缺少歌曲 ID"})
		return
	}

	playURL := "/api/download?id=" + url.QueryEscape(id) + "&play=1"
	if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" {
		playURL += "&name=" + url.QueryEscape(name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "url": playURL})
}
