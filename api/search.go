package handler

import (
	"encoding/json"
	"fmt"
	"handler/internal/provider/kuwo"
	"net/http"
)

func Search(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "请输入搜索关键词"})
		return
	}

	p := kuwo.New()
	songs, err := p.Search(keyword, 1, 30)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": fmt.Sprintf("搜索失败: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": songs})
}
