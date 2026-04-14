package server

import (
	"encoding/json"
	"fmt"
	"handler/internal/lol"
	"handler/internal/provider"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Server struct {
	provider provider.Provider
	lol      *lol.Client
	mux      *http.ServeMux
}

func New(p provider.Provider) *Server {
	s := &Server{provider: p, lol: lol.New(), mux: http.NewServeMux()}
	s.mux.HandleFunc("/api/search", s.handleSearch)
	s.mux.HandleFunc("/api/play", s.handlePlay)
	s.mux.HandleFunc("/api/download", s.handleDownload)
	s.mux.HandleFunc("/api/lol/query", s.handleLOLQuery)
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

	playURL := "/api/download?id=" + url.QueryEscape(id) + "&play=1"
	if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" {
		playURL += "&name=" + url.QueryEscape(name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "url": playURL})
}

// GET /api/download?id=xxx&name=xxx  代理下载
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("name")
	isPlay := r.URL.Query().Get("play") == "1"
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

	req, err := http.NewRequest(http.MethodGet, playURL, nil)
	if err != nil {
		jsonErr(w, "创建请求失败", 500)
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
		jsonErr(w, "下载失败", 500)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		if isPlay {
			http.Error(w, "播放失败", http.StatusBadGateway)
			return
		}
		jsonErr(w, "下载源返回异常", http.StatusBadGateway)
		return
	}

	for _, key := range []string{
		"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges",
	} {
		if value := resp.Header.Get(key); value != "" {
			w.Header().Set(key, value)
		}
	}

	// 过滤文件名非法字符
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

func (s *Server) handleLOLQuery(w http.ResponseWriter, r *http.Request) {
	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	areaName := strings.TrimSpace(r.URL.Query().Get("areaName"))
	if nickname == "" || areaName == "" {
		jsonErr(w, "缺少 nickname 或 areaName", 400)
		return
	}

	areaID, _ := strconv.Atoi(r.URL.Query().Get("areaId"))
	allCount, _ := strconv.Atoi(r.URL.Query().Get("allCount"))
	filter, _ := strconv.Atoi(r.URL.Query().Get("filter"))
	modelID, _ := strconv.Atoi(r.URL.Query().Get("modelId"))
	seleMe, _ := strconv.Atoi(r.URL.Query().Get("seleMe"))
	openID := strings.TrimSpace(r.URL.Query().Get("openId"))

	resp, err := s.lol.Query(lol.QueryParams{
		Nickname: nickname,
		AllCount: allCount,
		AreaID:   areaID,
		AreaName: areaName,
		SeleMe:   seleMe,
		Filter:   filter,
		OpenID:   openID,
		ModelID:  modelID,
	})
	if err != nil {
		jsonErr(w, fmt.Sprintf("LOL 查询失败: %v", err), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": resp})
}

func jsonErr(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": msg})
}
