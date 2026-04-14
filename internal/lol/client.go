package lol

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

const baseURL = "https://a.lzyumi.top/lzyumi/lol/info"

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type QueryParams struct {
	Nickname string
	AllCount int
	AreaID   int
	AreaName string
	SeleMe   int
	Filter   int
	OpenID   string
	ModelID  int
}

type Response struct {
	Code         int             `json:"code"`
	Message      string          `json:"message"`
	MessageDetail string         `json:"messageDetail"`
	PublicInfo   string          `json:"publicInfo"`
	OnlineInfo   string          `json:"onlineInfo"`
	BattleInfo   *BattleInfo     `json:"battleInfo"`
	Data         []MatchItem     `json:"data"`
	Raw          json.RawMessage `json:"-"`
}

type BattleInfo struct {
	OpenID      string         `json:"openId"`
	IconID      int            `json:"iconId"`
	Level       int            `json:"level"`
	NameInfoNew string         `json:"nameInfoNew"`
	Ext         string         `json:"ext"`
	Praise      string         `json:"praise"`
	AreaID      int            `json:"areaId"`
	LoadingImg  string         `json:"loadingImg"`
	Gender      string         `json:"mlolgender"`
	Location    string         `json:"mlollatestLocation"`
	SeasonInfo  map[string]any `json:"seasonInfoMap"`
}

type MatchItem struct {
	TitleTime       string `json:"titleTime"`
	Title           string `json:"title"`
	IsWin           int    `json:"isWin"`
	ChampionID      int    `json:"championId"`
	ChampionLevel   int    `json:"championIdLeve"`
	GameID          string `json:"gameId"`
	SkinURL         string `json:"skinUrl"`
	SummonSpell1ID  string `json:"summonSpell1Id"`
	SummonSpell2ID  string `json:"summonSpell2Id"`
	PerkStyleURL    string `json:"perkStyleUrl"`
	DamageInfo      string `json:"damageInfo"`
	GoldInfo        string `json:"goldInfo"`
}

func generateSigns(now time.Time) (string, string) {
	sm := strconv.Itoa(int(now.Month()))
	sd := strconv.Itoa(now.Day())
	sh := strconv.Itoa(now.Hour())
	smi := strconv.Itoa(now.Minute())
	ss := strconv.Itoa(now.Second())

	md5Str := fmt.Sprintf("dld%so%su%sd%so%sdld", pad2(sm), pad2(sd), pad2(sh), pad2(smi), pad2(ss))
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

func pad2(s string) string {
	if len(s) >= 2 {
		return s
	}
	return "0" + s
}

func sanitizeNickname(n string) string {
	return strings.ReplaceAll(strings.TrimSpace(n), "#", "*~*~*")
}

func (c *Client) Query(params QueryParams) (*Response, error) {
	if params.Nickname == "" {
		return nil, fmt.Errorf("nickname is required")
	}
	if params.AllCount <= 0 {
		params.AllCount = 10
	}
	if params.SeleMe == 0 {
		params.SeleMe = 1
	}
	if params.Filter == 0 {
		params.Filter = 1
	}
	if params.ModelID == 0 {
		params.ModelID = 1
	}
	if params.AreaID == 0 || params.AreaName == "" {
		return nil, fmt.Errorf("areaId and areaName are required")
	}

	lzyumiSign, signStr := generateSigns(time.Now())
	q := url.Values{}
	q.Set("nickname", sanitizeNickname(params.Nickname))
	q.Set("allCount", strconv.Itoa(params.AllCount))
	q.Set("areaId", strconv.Itoa(params.AreaID))
	q.Set("areaName", params.AreaName)
	q.Set("seleMe", strconv.Itoa(params.SeleMe))
	q.Set("filter", strconv.Itoa(params.Filter))
	q.Set("openId", params.OpenID)
	q.Set("modelId", strconv.Itoa(params.ModelID))
	q.Set("lzyumiSign", lzyumiSign)
	q.Set("signStr", signStr)

	req, err := http.NewRequest(http.MethodGet, baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://a.lzyumi.top/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
