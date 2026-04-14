package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const lolQueryBaseURL = "https://a.lzyumi.top/lzyumi/lol/info"

func Handler(w http.ResponseWriter, r *http.Request) {
	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	areaName := strings.TrimSpace(r.URL.Query().Get("areaName"))
	if nickname == "" || areaName == "" {
		writeLOLErr(w, 400, "缺少 nickname 或 areaName")
		return
	}

	areaID, _ := strconv.Atoi(r.URL.Query().Get("areaId"))
	allCount, _ := strconv.Atoi(r.URL.Query().Get("allCount"))
	filter, _ := strconv.Atoi(r.URL.Query().Get("filter"))
	modelID, _ := strconv.Atoi(r.URL.Query().Get("modelId"))
	seleMe, _ := strconv.Atoi(r.URL.Query().Get("seleMe"))
	openID := strings.TrimSpace(r.URL.Query().Get("openId"))

	if areaID == 0 {
		writeLOLErr(w, 400, "缺少或非法 areaId")
		return
	}
	if allCount <= 0 {
		allCount = 20
	}
	if filter == 0 {
		filter = 1
	}
	if modelID == 0 {
		modelID = 1
	}
	if seleMe == 0 {
		seleMe = 1
	}

	lzyumiSign, signStr := generateLOLSigns(time.Now())
	q := url.Values{}
	q.Set("nickname", strings.ReplaceAll(nickname, "#", "*~*~*"))
	q.Set("allCount", strconv.Itoa(allCount))
	q.Set("areaId", strconv.Itoa(areaID))
	q.Set("areaName", areaName)
	q.Set("seleMe", strconv.Itoa(seleMe))
	q.Set("filter", strconv.Itoa(filter))
	q.Set("openId", openID)
	q.Set("modelId", strconv.Itoa(modelID))
	q.Set("lzyumiSign", lzyumiSign)
	q.Set("signStr", signStr)

	req, err := http.NewRequest(http.MethodGet, lolQueryBaseURL+"?"+q.Encode(), nil)
	if err != nil {
		writeLOLErr(w, 500, "创建 LOL 请求失败")
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://a.lzyumi.top/")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeLOLErr(w, 500, fmt.Sprintf("LOL 查询失败: %v", err))
		return
	}
	defer resp.Body.Close()

	var out any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		writeLOLErr(w, 500, fmt.Sprintf("LOL 返回解析失败: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": out})
}

func generateLOLSigns(now time.Time) (string, string) {
	sm := strconv.Itoa(int(now.Month()))
	sd := strconv.Itoa(now.Day())
	sh := strconv.Itoa(now.Hour())
	smi := strconv.Itoa(now.Minute())
	ss := strconv.Itoa(now.Second())

	md5Str := fmt.Sprintf("dld%so%su%sd%so%sdld", padLOL2(sm), padLOL2(sd), padLOL2(sh), padLOL2(smi), padLOL2(ss))
	h := md5.Sum([]byte(md5Str))
	lzyumiSign := hex.EncodeToString(h[:])
	signStr := sm + sd + sh + smi + ss +
		strconv.Itoa(len(sm)*3) +
		strconv.Itoa(len(sd)*3) +
		strconv.Itoa(len(sh)*3) +
		strconv.Itoa(len(smi)*3) +
		strconv.Itoa(len(ss)*3)
	return lzyumiSign, signStr
}

func padLOL2(s string) string {
	if len(s) >= 2 {
		return s
	}
	return "0" + s
}

func writeLOLErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": msg})
}
