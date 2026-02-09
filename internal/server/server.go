package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"music-fetcher/internal/provider"
	"net/http"
	"strings"
)

type Server struct {
	provider provider.Provider
	mux      *http.ServeMux
}

func New(p provider.Provider) *Server {
	s := &Server{provider: p, mux: http.NewServeMux()}
	s.mux.HandleFunc("/api/search", s.handleSearch)
	s.mux.HandleFunc("/api/play", s.handlePlay)
	s.mux.HandleFunc("/api/download", s.handleDownload)
	s.mux.Handle("/", http.FileServer(http.Dir("static")))
	return s
}

func (s *Server) Start(addr string) error {
	log.Printf("Web 服务启动: http://%s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

// GET /api/search?keyword=xxx
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		jsonErr(w, "请输入搜索关键词", 400)
		return
	}

	songs, err := s.provider.Search(keyword, 1, 30)
	if err != nil {
		jsonErr(w, fmt.Sprintf("搜索失败: %v", err), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": songs})
}

// GET /api/play?id=xxx  返回播放链接
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonErr(w, "缺少歌曲 ID", 400)
		return
	}

	url, err := s.provider.GetPlayURL(id)
	if err != nil {
		jsonErr(w, fmt.Sprintf("获取链接失败: %v", err), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "url": url})
}

// GET /api/download?id=xxx&name=xxx  代理下载
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("name")
	if id == "" {
		jsonErr(w, "缺少歌曲 ID", 400)
		return
	}
	if name == "" {
		name = id
	}

	playURL, err := s.provider.GetPlayURL(id)
	if err != nil {
		jsonErr(w, fmt.Sprintf("获取链接失败: %v", err), 500)
		return
	}

	resp, err := http.Get(playURL)
	if err != nil {
		jsonErr(w, "下载失败", 500)
		return
	}
	defer resp.Body.Close()

	// 过滤文件名非法字符
	safeName := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\"", "_").Replace(name)

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.mp3"`, safeName))
	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}
	io.Copy(w, resp.Body)
}

func jsonErr(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": msg})
}
