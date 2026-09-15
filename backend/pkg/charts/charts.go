package charts

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	reKuwoKwcdn = regexp.MustCompile(`^http://img\d*\.kwcdn\.kuwo\.cn`)
	reKuwoImg   = regexp.MustCompile(`^http://img\d*\.kuwo\.cn`)
)

func fixKuwoCover(pic, pic2, pic5 string) string {
	cover := pic2
	if cover == "" || !strings.HasPrefix(cover, "https://") {
		if strings.HasPrefix(pic, "https://") {
			cover = pic
		} else if strings.HasPrefix(pic5, "https://") {
			cover = pic5
		} else if pic != "" {
			cover = pic
		}
	}
	if reKuwoKwcdn.MatchString(cover) {
		cover = reKuwoKwcdn.ReplaceAllString(cover, "https://kwimg1.kuwo.cn")
	} else if reKuwoImg.MatchString(cover) {
		cover = reKuwoImg.ReplaceAllString(cover, "https://kwimg1.kuwo.cn")
	} else if strings.HasPrefix(cover, "http://") {
		cover = "https://" + strings.TrimPrefix(cover, "http://")
	}
	return cover
}

func cleanSourceName(name string) string {
	name = strings.ReplaceAll(name, "网易云音乐", "wy")
	name = strings.ReplaceAll(name, "网易云", "wy")
	name = strings.ReplaceAll(name, "网易", "wy")
	name = strings.ReplaceAll(name, "酷我音乐", "kw")
	name = strings.ReplaceAll(name, "酷我", "kw")
	name = strings.ReplaceAll(name, "酷狗音乐", "kg")
	name = strings.ReplaceAll(name, "酷狗", "kg")
	name = strings.ReplaceAll(name, "QQ音乐", "tx")
	name = strings.ReplaceAll(name, "腾讯音乐", "tx")
	return name
}

func toInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func toInt(val interface{}) int {
	return int(toInt64(val))
}

type ToplistItem struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Cover           string   `json:"cover"`
	UpdateFrequency string   `json:"update_frequency"`
	PlayCount       int64    `json:"play_count"`
	Tracks          []string `json:"tracks"`
	Source          string   `json:"source"`
}

type ChartSongItem struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Singer   string   `json:"singer"`
	Album    string   `json:"album"`
	Duration int      `json:"duration"`
	Interval int      `json:"interval"`
	Cover    string   `json:"cover"`
	Source   string   `json:"source"`
	Songmid  string   `json:"songmid"`
	Hash     string   `json:"hash,omitempty"`
	Quality  string   `json:"quality,omitempty"`
	Qualitys []string `json:"qualitys,omitempty"`
}

type PlaylistItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	PlayCount   int64  `json:"play_count"`
	TrackCount  int    `json:"track_count"`
	Creator     string `json:"creator"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type ChartManager struct {
	client     *http.Client
	mu         sync.RWMutex
	topCache   map[string][]ToplistItem
	topCacheAt map[string]time.Time
}

func NewChartManager() *ChartManager {
	return &ChartManager{
		client:     &http.Client{Timeout: 10 * time.Second},
		topCache:   make(map[string][]ToplistItem),
		topCacheAt: make(map[string]time.Time),
	}
}

// ── 1. 排行榜单 ──

func (cm *ChartManager) HandleToplists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = "wy"
	}

	cm.mu.RLock()
	if cached, ok := cm.topCache[source]; ok && time.Since(cm.topCacheAt[source]) < 15*time.Minute {
		cm.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": cached,
		})
		return
	}
	cm.mu.RUnlock()

	var items []ToplistItem
	var err error

	switch source {
	case "kg":
		items, err = cm.fetchKugouToplists()
	case "tx":
		items, err = cm.fetchQQToplists()
	case "kw":
		items, err = cm.fetchKuwoToplists()
	case "wy":
		items, err = cm.fetchNeteaseToplists()
	default:
		items = []ToplistItem{}
	}

	if err != nil {
		http.Error(w, "Failed to fetch toplists: "+err.Error(), http.StatusBadGateway)
		return
	}

	cm.mu.Lock()
	cm.topCache[source] = items
	cm.topCacheAt[source] = time.Now()
	cm.mu.Unlock()

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": items,
	})
}

func (cm *ChartManager) fetchKuwoToplists() ([]ToplistItem, error) {
	req, err := http.NewRequest("GET", "http://wapi.kuwo.cn/api/pc/bang/list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Child []struct {
			Name  string `json:"name"`
			Child []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Pic      string `json:"pic"`
				Pic2     string `json:"pic2"`
				Pic5     string `json:"pic5"`
				SourceID string `json:"sourceid"`
				Intro    string `json:"intro"`
			} `json:"child"`
		} `json:"child"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	items := make([]ToplistItem, 0)
	for _, cat := range raw.Child {
		for _, b := range cat.Child {
			idInt, _ := strconv.ParseInt(b.SourceID, 10, 64)
			if idInt == 0 {
				idInt, _ = strconv.ParseInt(b.ID, 10, 64)
			}
			cover := fixKuwoCover(b.Pic, b.Pic2, b.Pic5)
			items = append(items, ToplistItem{
				ID:              idInt,
				Name:            cleanSourceName(b.Name),
				Cover:           cover,
				UpdateFrequency: cleanSourceName(cat.Name),
				PlayCount:       0,
				Tracks:          []string{},
				Source:          "kw",
			})
		}
	}
	return items, nil
}

func (cm *ChartManager) fetchKugouToplists() ([]ToplistItem, error) {
	req, err := http.NewRequest("GET", "http://m.kugou.com/rank/list&json=true", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Rank struct {
			List []struct {
				RankID   int64  `json:"rankid"`
				RankName string `json:"rankname"`
				ImgURL   string `json:"imgurl"`
			} `json:"list"`
		} `json:"rank"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	items := make([]ToplistItem, 0, len(raw.Rank.List))
	for _, it := range raw.Rank.List {
		cover := strings.ReplaceAll(it.ImgURL, "{size}", "240")
		items = append(items, ToplistItem{
			ID:              it.RankID,
			Name:            cleanSourceName(it.RankName),
			Cover:           cover,
			UpdateFrequency: "官方更新",
			PlayCount:       0,
			Tracks:          []string{},
			Source:          "kg",
		})
	}
	return items, nil
}

func (cm *ChartManager) fetchQQToplists() ([]ToplistItem, error) {
	req, err := http.NewRequest("GET", "https://c.y.qq.com/v8/fcg-bin/fcg_myqq_toplist.fcg?format=json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Data struct {
			TopList []struct {
				ID          int64  `json:"id"`
				TopTitle    string `json:"topTitle"`
				PicURL      string `json:"picUrl"`
				ListenCount int64  `json:"listenCount"`
				SongList    []struct {
					Songname   string `json:"songname"`
					Singername string `json:"singername"`
				} `json:"songList"`
			} `json:"topList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	items := make([]ToplistItem, 0, len(raw.Data.TopList))
	for _, it := range raw.Data.TopList {
		tracks := make([]string, 0)
		for _, s := range it.SongList {
			tracks = append(tracks, fmt.Sprintf("%s - %s", s.Songname, s.Singername))
		}
		items = append(items, ToplistItem{
			ID:              it.ID,
			Name:            cleanSourceName(it.TopTitle),
			Cover:           it.PicURL,
			UpdateFrequency: "官方更新",
			PlayCount:       it.ListenCount,
			Tracks:          tracks,
			Source:          "tx",
		})
	}
	return items, nil
}

func (cm *ChartManager) fetchNeteaseToplists() ([]ToplistItem, error) {
	req, err := http.NewRequest("GET", "https://music.163.com/api/toplist", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://music.163.com/")

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Code int `json:"code"`
		List []struct {
			ID              int64  `json:"id"`
			Name            string `json:"name"`
			CoverImgUrl     string `json:"coverImgUrl"`
			UpdateFrequency string `json:"updateFrequency"`
			PlayCount       int64  `json:"playCount"`
			Tracks          []struct {
				First  string `json:"first"`
				Second string `json:"second"`
			} `json:"tracks"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	items := make([]ToplistItem, 0, len(raw.List))
	for _, it := range raw.List {
		tracks := make([]string, 0)
		for _, t := range it.Tracks {
			tracks = append(tracks, fmt.Sprintf("%s - %s", t.First, t.Second))
		}
		items = append(items, ToplistItem{
			ID:              it.ID,
			Name:            cleanSourceName(it.Name),
			Cover:           it.CoverImgUrl,
			UpdateFrequency: cleanSourceName(it.UpdateFrequency),
			PlayCount:       it.PlayCount,
			Tracks:          tracks,
			Source:          "wy",
		})
	}
	return items, nil
}

// ── 2. 榜单详情 ──

func (cm *ChartManager) HandleChartDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = "wy"
	}
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		http.Error(w, "id parameter required", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))

	switch source {
	case "kg":
		cm.fetchAndWriteKugouChartDetail(w, idStr)
	case "tx":
		cm.fetchAndWriteQQChartDetail(w, idStr)
	case "kw":
		cm.fetchAndWriteKuwoChartDetail(w, idStr, name)
	case "wy":
		cm.fetchAndWriteNeteasePlaylistDetail(w, idStr)
	default:
		http.Error(w, "Unsupported source", http.StatusBadRequest)
	}
}

func (cm *ChartManager) getKuwoChartName(bangId string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	idInt, _ := strconv.ParseInt(bangId, 10, 64)
	if cached, ok := cm.topCache["kw"]; ok {
		for _, it := range cached {
			if it.ID == idInt {
				return cleanSourceName(it.Name)
			}
		}
	}
	switch bangId {
	case "16":
		return "kw热歌榜"
	case "93":
		return "kw飙升榜"
	case "17":
		return "kw新歌榜"
	case "104":
		return "kw华语榜"
	case "182":
		return "kw粤语榜"
	case "22":
		return "kw欧美榜"
	case "328":
		return "车载歌曲榜"
	case "297":
		return "跑步健身榜"
	case "255":
		return "KTV点唱榜"
	default:
		return "kw热歌榜"
	}
}

func (cm *ChartManager) fetchKuwoCover(rid string) string {
	if rid == "" || rid == "0" {
		return ""
	}
	u := fmt.Sprintf("http://artistpicserver.kuwo.cn/pic.web?corp=kuwo&type=rid_pic&pictype=url&size=240&rid=%s", rid)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := cm.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(b))
	if text == "" || text == "NO_PIC" || !strings.HasPrefix(text, "http") {
		return ""
	}
	return fixKuwoCover(text, "", "")
}

func (cm *ChartManager) fetchAndWriteKuwoChartDetail(w http.ResponseWriter, bangId, bangName string) {
	if bangName == "" {
		bangName = cm.getKuwoChartName(bangId)
	}
	bangName = cleanSourceName(bangName)

	apiURL := fmt.Sprintf("http://kbangserver.kuwo.cn/ksong.s?from=pc&fmt=json&type=bang&data=content&id=%s&pn=0&rn=100", bangId)
	req, err := http.NewRequest("GET", apiURL, nil)
	var songs []ChartSongItem
	var chartCover string
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		if resp, err := cm.client.Do(req); err == nil {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				var raw struct {
					Name      string `json:"name"`
					Pic       string `json:"pic"`
					V9Pic2    string `json:"v9_pic2"`
					Musiclist []struct {
						ID           interface{} `json:"id"`
						Name         string      `json:"name"`
						Artist       string      `json:"artist"`
						Album        string      `json:"album"`
						Duration     interface{} `json:"duration"`
						SongDuration interface{} `json:"song_duration"`
					} `json:"musiclist"`
				}
				if json.Unmarshal(body, &raw) == nil && len(raw.Musiclist) > 0 {
					if raw.Name != "" {
						bangName = cleanSourceName(raw.Name)
					}
					chartCover = fixKuwoCover(raw.Pic, raw.V9Pic2, "")
					for _, m := range raw.Musiclist {
						idStr := strings.TrimSpace(fmt.Sprintf("%v", m.ID))
						if idStr == "" || idStr == "0" {
							continue
						}
						dur := toInt(m.SongDuration)
						if dur <= 0 {
							dur = toInt(m.Duration)
						}
						if dur <= 0 {
							dur = 180
						}
						sName := cleanSourceName(html.UnescapeString(m.Name))
						sName = strings.ReplaceAll(sName, "&nbsp;", " ")
						songs = append(songs, ChartSongItem{
							ID:       "kw_" + idStr,
							Name:     sName,
							Singer:   cleanSourceName(html.UnescapeString(m.Artist)),
							Album:    cleanSourceName(html.UnescapeString(m.Album)),
							Duration: dur,
							Interval: dur,
							Cover:    "",
							Source:   "kw",
							Songmid:  idStr,
						})
					}
				}
			}
		}
	}

	// Fallback to search if kbangserver returned no songs
	if len(songs) == 0 {
		songs = cm.searchKuwoSongs(bangName)
	} else {
		// 并发拉取单曲高保真专辑/歌手封面 (限制并发度 15，极速 ~100ms 完成)
		var wg sync.WaitGroup
		sem := make(chan struct{}, 15)
		for i := range songs {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				c := cm.fetchKuwoCover(songs[idx].Songmid)
				if c != "" {
					songs[idx].Cover = c
				} else if chartCover != "" {
					songs[idx].Cover = chartCover
				}
			}(i)
		}
		wg.Wait()
	}

	idInt, _ := strconv.ParseInt(bangId, 10, 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        bangName,
			"cover":       chartCover,
			"description": "kw官方排行榜单",
			"play_count":  0,
			"track_count": len(songs),
			"creator":     "kw",
			"songs":       songs,
		},
	})
}

func (cm *ChartManager) searchKuwoSongs(keyword string) []ChartSongItem {
	apiURL := fmt.Sprintf("https://search.kuwo.cn/r.s?all=%s&ft=music&itemset=web_2013&client=kt&pn=0&rn=100&rformat=json&encoding=utf8", url.QueryEscape(keyword))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	s := strings.ReplaceAll(string(body), "'", "\"")
	var raw struct {
		Abslist []struct {
			Artist            string `json:"ARTIST"`
			Songname          string `json:"SONGNAME"`
			Album             string `json:"ALBUM"`
			Duration          string `json:"DURATION"`
			DcTargetId        string `json:"DC_TARGETID"`
			MusicRid          string `json:"MUSICRID"`
			WebAlbumpicShort  string `json:"web_albumpic_short"`
			WebArtistpicShort string `json:"web_artistpic_short"`
		} `json:"abslist"`
	}

	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil
	}

	songs := make([]ChartSongItem, 0, len(raw.Abslist))
	for _, item := range raw.Abslist {
		rid := item.DcTargetId
		if rid == "" {
			rid = strings.TrimPrefix(item.MusicRid, "MUSIC_")
		}
		if rid == "" {
			continue
		}
		dur, _ := strconv.Atoi(item.Duration)
		if dur <= 0 {
			dur = 180
		}
		name := html.UnescapeString(item.Songname)
		name = strings.ReplaceAll(name, "&nbsp;", " ")

		cover := ""
		if item.WebAlbumpicShort != "" {
			hdPath := strings.Replace(item.WebAlbumpicShort, "120/", "500/", 1)
			cover = fixKuwoCover("https://img1.kuwo.cn/star/albumcover/"+hdPath, "", "")
		} else if item.WebArtistpicShort != "" {
			cover = fixKuwoCover("https://img1.kuwo.cn/star/starheads/"+item.WebArtistpicShort, "", "")
		}

		songs = append(songs, ChartSongItem{
			ID:       "kw_" + rid,
			Name:     cleanSourceName(name),
			Singer:   cleanSourceName(html.UnescapeString(item.Artist)),
			Album:    cleanSourceName(html.UnescapeString(item.Album)),
			Duration: dur,
			Interval: dur,
			Cover:    cover,
			Source:   "kw",
			Songmid:  rid,
		})
	}
	return songs
}

func (cm *ChartManager) fetchAndWriteKugouChartDetail(w http.ResponseWriter, rankId string) {
	var allSongs []ChartSongItem
	var rankTitle string
	var bannerUrl string

	for page := 1; page <= 3; page++ {
		apiURL := fmt.Sprintf("http://m.kugou.com/rank/info/?rankid=%s&page=%d&json=true", rankId, page)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")

		resp, err := cm.client.Do(req)
		if err != nil {
			break
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var raw struct {
			Info struct {
				RankName   string `json:"rankname"`
				Banner7URL string `json:"banner7url"`
			} `json:"info"`
			Songs struct {
				List []struct {
					FileName          string `json:"filename"`
					SongName          string `json:"songname"`
					SingerName        string `json:"singername"`
					Hash              string `json:"hash"`
					Duration          int    `json:"duration"`
					AlbumSizableCover string `json:"album_sizable_cover"`
					TransParam        struct {
						UnionCover string `json:"union_cover"`
					} `json:"trans_param"`
				} `json:"list"`
			} `json:"songs"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			break
		}

		if rankTitle == "" {
			rankTitle = raw.Info.RankName
		}
		if bannerUrl == "" {
			bannerUrl = raw.Info.Banner7URL
		}

		if len(raw.Songs.List) == 0 {
			break
		}

		for _, s := range raw.Songs.List {
			songName := s.SongName
			singer := s.SingerName
			if singer == "" && strings.Contains(s.FileName, " - ") {
				parts := strings.SplitN(s.FileName, " - ", 2)
				singer = strings.TrimSpace(parts[0])
				if songName == "" {
					songName = strings.TrimSpace(parts[1])
				}
			}
			if songName == "" {
				songName = s.FileName
			}
			if singer == "" {
				singer = "群星"
			}
			dur := s.Duration
			if dur <= 0 {
				dur = 180
			}

			songCover := s.AlbumSizableCover
			if songCover == "" {
				songCover = s.TransParam.UnionCover
			}
			if songCover != "" {
				songCover = strings.ReplaceAll(songCover, "{size}", "240")
				if strings.HasPrefix(songCover, "http://") {
					songCover = strings.Replace(songCover, "http://", "https://", 1)
				}
			} else {
				songCover = strings.ReplaceAll(bannerUrl, "{size}", "240")
			}

			allSongs = append(allSongs, ChartSongItem{
				ID:       "kg_" + s.Hash,
				Name:     cleanSourceName(songName),
				Singer:   cleanSourceName(singer),
				Album:    "",
				Duration: dur,
				Interval: dur,
				Cover:    songCover,
				Source:   "kg",
				Songmid:  s.Hash,
				Hash:     s.Hash,
			})
		}
	}

	cover := strings.ReplaceAll(bannerUrl, "{size}", "240")
	idInt, _ := strconv.ParseInt(rankId, 10, 64)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        cleanSourceName(rankTitle),
			"cover":       cover,
			"description": "kg官方权威排行榜",
			"play_count":  0,
			"track_count": len(allSongs),
			"creator":     "kg",
			"songs":       allSongs,
		},
	})
}

func (cm *ChartManager) fetchAndWriteQQChartDetail(w http.ResponseWriter, topId string) {
	apiURL := fmt.Sprintf("https://c.y.qq.com/v8/fcg-bin/fcg_v8_toplist_cp.fcg?topid=%s&format=json", topId)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch QQ chart: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Topinfo struct {
			ListName string `json:"ListName"`
			PicAlbum string `json:"pic_album"`
		} `json:"topinfo"`
		Songlist []struct {
			Data struct {
				Songname  string `json:"songname"`
				Songmid   string `json:"songmid"`
				Albumname string `json:"albumname"`
				Albummid  string `json:"albummid"`
				Interval  int    `json:"interval"`
				Singer    []struct {
					Name string `json:"name"`
				} `json:"singer"`
			} `json:"data"`
		} `json:"songlist"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid QQ chart JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	songs := make([]ChartSongItem, 0, len(raw.Songlist))
	for _, it := range raw.Songlist {
		s := it.Data
		singerNames := make([]string, 0, len(s.Singer))
		for _, a := range s.Singer {
			singerNames = append(singerNames, a.Name)
		}
		singer := strings.Join(singerNames, ", ")
		if singer == "" {
			singer = "群星"
		}
		cover := ""
		if s.Albummid != "" {
			cover = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", s.Albummid)
		} else {
			cover = raw.Topinfo.PicAlbum
		}
		dur := s.Interval
		if dur <= 0 {
			dur = 180
		}

		songs = append(songs, ChartSongItem{
			ID:       "tx_" + s.Songmid,
			Name:     s.Songname,
			Singer:   singer,
			Album:    s.Albumname,
			Duration: dur,
			Interval: dur,
			Cover:    cover,
			Source:   "tx",
			Songmid:  s.Songmid,
		})
	}

	idInt, _ := strconv.ParseInt(topId, 10, 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        cleanSourceName(raw.Topinfo.ListName),
			"cover":       raw.Topinfo.PicAlbum,
			"description": "tx官方巅峰排行榜",
			"play_count":  0,
			"track_count": len(songs),
			"creator":     "tx",
			"songs":       songs,
		},
	})
}

// ── 3. 精选歌单 ──

func (cm *ChartManager) HandlePlaylists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = "wy"
	}
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "全部"
	}
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 24
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	switch source {
	case "kg":
		cm.fetchAndWriteKugouPlaylists(w, page, limit)
	case "tx":
		cm.fetchAndWriteQQPlaylists(w, page, limit)
	case "kw":
		cm.fetchAndWriteKuwoPlaylists(w, page, limit)
	case "wy":
		cm.fetchAndWriteNeteasePlaylists(w, category, page, limit)
	default:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":     200,
			"category": category,
			"page":     page,
			"limit":    limit,
			"total":    0,
			"data":     []PlaylistItem{},
		})
	}
}

func (cm *ChartManager) fetchAndWriteKugouPlaylists(w http.ResponseWriter, page, limit int) {
	req, err := http.NewRequest("GET", "http://m.kugou.com/plist/index&json=true", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch Kugou playlists: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Plist struct {
			List struct {
				Info []struct {
					SpecialID   int64  `json:"specialid"`
					SpecialName string `json:"specialname"`
					ImgURL      string `json:"imgurl"`
					PlayCount   int64  `json:"playcount"`
					SongCount   int    `json:"songcount"`
					Intro       string `json:"intro"`
				} `json:"info"`
				Total int `json:"total"`
			} `json:"list"`
		} `json:"plist"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid Kugou playlist JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]PlaylistItem, 0, len(raw.Plist.List.Info))
	for _, p := range raw.Plist.List.Info {
		cover := strings.ReplaceAll(p.ImgURL, "{size}", "240")
		cover = strings.Replace(cover, "http://", "https://", 1)
		items = append(items, PlaylistItem{
			ID:          p.SpecialID,
			Title:       cleanSourceName(p.SpecialName),
			Cover:       cover,
			PlayCount:   p.PlayCount,
			TrackCount:  p.SongCount,
			Creator:     "kg精选",
			Description: cleanSourceName(p.Intro),
			Source:      "kg",
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":     200,
		"category": "kg精选",
		"page":     page,
		"limit":    limit,
		"total":    raw.Plist.List.Total,
		"data":     items,
	})
}

func (cm *ChartManager) fetchAndWriteKuwoPlaylists(w http.ResponseWriter, page, limit int) {
	pn := page - 1
	if pn < 0 {
		pn = 0
	}
	apiURL := fmt.Sprintf("http://wapi.kuwo.cn/api/pc/classify/playlist/getRcmPlayList?pn=%d&rn=%d&order=hot", pn, limit)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch Kuwo playlists: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Code int `json:"code"`
		Data struct {
			Total interface{} `json:"total"`
			Data  []struct {
				ID        interface{} `json:"id"`
				Name      string      `json:"name"`
				Img       string      `json:"img"`
				Uname     string      `json:"uname"`
				Total     interface{} `json:"total"`
				Listencnt interface{} `json:"listencnt"`
				Info      string      `json:"info"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid Kuwo playlists JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]PlaylistItem, 0, len(raw.Data.Data))
	for _, p := range raw.Data.Data {
		cover := fixKuwoCover(p.Img, "", "")
		idVal := toInt64(p.ID)
		items = append(items, PlaylistItem{
			ID:          idVal,
			Title:       cleanSourceName(html.UnescapeString(p.Name)),
			Cover:       cover,
			PlayCount:   toInt64(p.Listencnt),
			TrackCount:  toInt(p.Total),
			Creator:     cleanSourceName(p.Uname),
			Description: cleanSourceName(p.Info),
			Source:      "kw",
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":     200,
		"category": "kw精选",
		"page":     page,
		"limit":    limit,
		"total":    toInt(raw.Data.Total),
		"data":     items,
	})
}

func (cm *ChartManager) fetchAndWriteQQPlaylists(w http.ResponseWriter, page, limit int) {
	sin := (page - 1) * limit
	ein := sin + limit - 1
	apiURL := fmt.Sprintf("https://c.y.qq.com/splcloud/fcgi-bin/fcg_get_diss_by_tag.fcg?picmid=1&rnd=0.123&g_tk=5381&loginUin=0&hostUin=0&format=json&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0&categoryId=10000000&sortId=5&sin=%d&ein=%d", sin, ein)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://y.qq.com/")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch QQ playlists: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Code int `json:"code"`
		Data struct {
			Sum  int `json:"sum"`
			List []struct {
				Dissid    string `json:"dissid"`
				Dissname  string `json:"dissname"`
				Imgurl    string `json:"imgurl"`
				Listennum int64  `json:"listennum"`
				Creator   struct {
					Name string `json:"name"`
				} `json:"creator"`
			} `json:"list"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid QQ playlists JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]PlaylistItem, 0, len(raw.Data.List))
	for _, p := range raw.Data.List {
		idInt, _ := strconv.ParseInt(p.Dissid, 10, 64)
		cover := strings.Replace(p.Imgurl, "http://", "https://", 1)
		items = append(items, PlaylistItem{
			ID:          idInt,
			Title:       cleanSourceName(html.UnescapeString(p.Dissname)),
			Cover:       cover,
			PlayCount:   p.Listennum,
			TrackCount:  0,
			Creator:     cleanSourceName(p.Creator.Name),
			Description: "tx官方精选歌单",
			Source:      "tx",
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":     200,
		"category": "tx精选",
		"page":     page,
		"limit":    limit,
		"total":    raw.Data.Sum,
		"data":     items,
	})
}

func (cm *ChartManager) fetchAndWriteNeteasePlaylists(w http.ResponseWriter, category string, page, limit int) {
	offset := (page - 1) * limit
	apiURL := fmt.Sprintf("https://music.163.com/api/playlist/list?cat=%s&order=hot&offset=%d&limit=%d",
		url.QueryEscape(category), offset, limit)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://music.163.com/")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch playlists: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Code      int `json:"code"`
		Total     int `json:"total"`
		Playlists []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			CoverImgUrl string `json:"coverImgUrl"`
			PlayCount   int64  `json:"playCount"`
			TrackCount  int    `json:"trackCount"`
			Description string `json:"description"`
			Creator     struct {
				Nickname string `json:"nickname"`
			} `json:"creator"`
		} `json:"playlists"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid playlist JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]PlaylistItem, 0, len(raw.Playlists))
	for _, p := range raw.Playlists {
		items = append(items, PlaylistItem{
			ID:          p.ID,
			Title:       p.Name,
			Cover:       p.CoverImgUrl,
			PlayCount:   p.PlayCount,
			TrackCount:  p.TrackCount,
			Creator:     p.Creator.Nickname,
			Description: p.Description,
			Source:      "wy",
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":     200,
		"category": category,
		"page":     page,
		"limit":    limit,
		"total":    raw.Total,
		"data":     items,
	})
}

// ── 4. 歌单详情 ──

func (cm *ChartManager) HandlePlaylistDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = "wy"
	}
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		http.Error(w, "id parameter required", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))

	switch source {
	case "kg":
		cm.fetchAndWriteKugouPlaylistDetail(w, idStr)
	case "tx":
		cm.fetchAndWriteQQPlaylistDetail(w, idStr)
	case "kw":
		cm.fetchAndWriteKuwoPlaylistDetail(w, idStr, name)
	case "wy":
		cm.fetchAndWriteNeteasePlaylistDetail(w, idStr)
	default:
		http.Error(w, "Unsupported source", http.StatusBadRequest)
	}
}

func (cm *ChartManager) fetchAndWriteKuwoPlaylistDetail(w http.ResponseWriter, specialId, specialName string) {
	if specialName == "" {
		specialName = "精选歌曲"
	}
	specialName = cleanSourceName(specialName)

	apiURL := fmt.Sprintf("http://nplserver.kuwo.cn/pl.svc?op=getlistinfo&pid=%s&pn=0&rn=100&encode=utf-8&keyset=pl2012&identity=kuwo", specialId)
	req, err := http.NewRequest("GET", apiURL, nil)
	var songs []ChartSongItem
	var playlistCover string
	var playlistDesc string
	var creator string
	var playCount int64

	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		if resp, err := cm.client.Do(req); err == nil {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				var raw struct {
					Title     string      `json:"title"`
					Info      string      `json:"info"`
					Pic       string      `json:"pic"`
					Uname     string      `json:"uname"`
					Playnum   interface{} `json:"playnum"`
					Musiclist []struct {
						ID       interface{} `json:"id"`
						Name     string      `json:"name"`
						Artist   string      `json:"artist"`
						Album    string      `json:"album"`
						Duration interface{} `json:"duration"`
						Albumpic string      `json:"albumpic"`
					} `json:"musiclist"`
				}
				if json.Unmarshal(body, &raw) == nil && len(raw.Musiclist) > 0 {
					if raw.Title != "" {
						specialName = cleanSourceName(raw.Title)
					}
					playlistCover = fixKuwoCover(raw.Pic, "", "")
					playlistDesc = cleanSourceName(raw.Info)
					creator = cleanSourceName(raw.Uname)
					playCount = toInt64(raw.Playnum)

					for _, m := range raw.Musiclist {
						idStr := strings.TrimSpace(fmt.Sprintf("%v", m.ID))
						if idStr == "" || idStr == "0" {
							continue
						}
						dur := toInt(m.Duration)
						if dur <= 0 {
							dur = 180
						}
						sName := cleanSourceName(html.UnescapeString(m.Name))
						sName = strings.ReplaceAll(sName, "&nbsp;", " ")
						songs = append(songs, ChartSongItem{
							ID:       "kw_" + idStr,
							Name:     sName,
							Singer:   cleanSourceName(html.UnescapeString(m.Artist)),
							Album:    cleanSourceName(html.UnescapeString(m.Album)),
							Duration: dur,
							Interval: dur,
							Cover:    fixKuwoCover(m.Albumpic, "", ""),
							Source:   "kw",
							Songmid:  idStr,
						})
					}
				}
			}
		}
	}

	// Fallback to search if nplserver returned no songs
	if len(songs) == 0 {
		songs = cm.searchKuwoSongs(specialName)
	}

	if playlistDesc == "" {
		playlistDesc = "kw精选歌单曲目"
	}
	if creator == "" {
		creator = "kw精选"
	}

	idInt, _ := strconv.ParseInt(specialId, 10, 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        specialName,
			"cover":       playlistCover,
			"description": playlistDesc,
			"play_count":  playCount,
			"track_count": len(songs),
			"creator":     creator,
			"songs":       songs,
		},
	})
}

func (cm *ChartManager) fetchAndWriteQQPlaylistDetail(w http.ResponseWriter, dissid string) {
	apiURL := fmt.Sprintf("https://c.y.qq.com/qzone/fcg-bin/fcg_ucc_getcdinfo_byids_cp.fcg?type=1&json=1&utf8=1&onlysong=0&disstid=%s&format=json&g_tk=5381&loginUin=0&hostUin=0&inCharset=utf8&outCharset=utf-8&notice=0&platform=yqq.json&needNewCode=0", dissid)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://y.qq.com/")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch QQ playlist detail: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Code   int `json:"code"`
		Cdlist []struct {
			Disstid  string `json:"disstid"`
			Dissname string `json:"dissname"`
			Logo     string `json:"logo"`
			Desc     string `json:"desc"`
			Songlist []struct {
				Songname  string `json:"songname"`
				Songmid   string `json:"songmid"`
				Albumname string `json:"albumname"`
				Albummid  string `json:"albummid"`
				Interval  int    `json:"interval"`
				Singer    []struct {
					Name string `json:"name"`
				} `json:"singer"`
			} `json:"songlist"`
		} `json:"cdlist"`
	}

	if err := json.Unmarshal(body, &raw); err != nil || len(raw.Cdlist) == 0 {
		http.Error(w, "Failed to parse QQ playlist detail", http.StatusInternalServerError)
		return
	}

	cd := raw.Cdlist[0]
	cover := strings.Replace(cd.Logo, "http://", "https://", 1)
	songs := make([]ChartSongItem, 0, len(cd.Songlist))

	for _, s := range cd.Songlist {
		singerNames := make([]string, 0, len(s.Singer))
		for _, a := range s.Singer {
			singerNames = append(singerNames, a.Name)
		}
		singer := strings.Join(singerNames, ", ")
		if singer == "" {
			singer = "群星"
		}
		scover := ""
		if s.Albummid != "" {
			scover = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", s.Albummid)
		} else {
			scover = cover
		}
		dur := s.Interval
		if dur <= 0 {
			dur = 180
		}

		songs = append(songs, ChartSongItem{
			ID:       "tx_" + s.Songmid,
			Name:     html.UnescapeString(s.Songname),
			Singer:   singer,
			Album:    html.UnescapeString(s.Albumname),
			Duration: dur,
			Interval: dur,
			Cover:    scover,
			Source:   "tx",
			Songmid:  s.Songmid,
		})
	}

	idInt, _ := strconv.ParseInt(dissid, 10, 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        cleanSourceName(html.UnescapeString(cd.Dissname)),
			"cover":       cover,
			"description": cleanSourceName(cd.Desc),
			"play_count":  0,
			"track_count": len(songs),
			"creator":     "tx",
			"songs":       songs,
		},
	})
}

func (cm *ChartManager) fetchAndWriteKugouPlaylistDetail(w http.ResponseWriter, specialId string) {
	apiURL := fmt.Sprintf("http://m.kugou.com/plist/list/%s?json=true", specialId)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch Kugou playlist detail: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Info struct {
			List struct {
				SpecialName string `json:"specialname"`
				ImgURL      string `json:"imgurl"`
				Intro       string `json:"intro"`
			} `json:"list"`
		} `json:"info"`
		List struct {
			List struct {
				Info []struct {
					FileName   string `json:"filename"`
					SongName   string `json:"songname"`
					SingerName string `json:"singername"`
					Hash       string `json:"hash"`
					Duration   int    `json:"duration"`
				} `json:"info"`
				Total int `json:"total"`
			} `json:"list"`
		} `json:"list"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid Kugou playlist detail JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cover := strings.ReplaceAll(raw.Info.List.ImgURL, "{size}", "240")
	songs := make([]ChartSongItem, 0, len(raw.List.List.Info))

	for _, s := range raw.List.List.Info {
		songName := s.SongName
		singer := s.SingerName
		if singer == "" && strings.Contains(s.FileName, " - ") {
			parts := strings.SplitN(s.FileName, " - ", 2)
			singer = strings.TrimSpace(parts[0])
			if songName == "" {
				songName = strings.TrimSpace(parts[1])
			}
		}
		if songName == "" {
			songName = s.FileName
		}
		if singer == "" {
			singer = "群星"
		}
		dur := s.Duration
		if dur <= 0 {
			dur = 180
		}

		songs = append(songs, ChartSongItem{
			ID:       "kg_" + s.Hash,
			Name:     songName,
			Singer:   singer,
			Album:    "",
			Duration: dur,
			Interval: dur,
			Cover:    cover,
			Source:   "kg",
			Songmid:  s.Hash,
			Hash:     s.Hash,
		})
	}

	idInt, _ := strconv.ParseInt(specialId, 10, 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          idInt,
			"name":        cleanSourceName(raw.Info.List.SpecialName),
			"cover":       cover,
			"description": cleanSourceName(raw.Info.List.Intro),
			"play_count":  0,
			"track_count": len(songs),
			"creator":     "kg精选",
			"songs":       songs,
		},
	})
}

func (cm *ChartManager) fetchAndWriteNeteasePlaylistDetail(w http.ResponseWriter, idStr string) {
	apiURL := fmt.Sprintf("https://music.163.com/api/playlist/detail?id=%s&s=0&n=1000", idStr)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("Cookie", "os=pc; osver=Microsoft-Windows-10; appver=2.9.7;")

	resp, err := cm.client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch playlist detail: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Code   int `json:"code"`
		Result struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			CoverImgUrl string `json:"coverImgUrl"`
			Description string `json:"description"`
			PlayCount   int64  `json:"playCount"`
			TrackCount  int    `json:"trackCount"`
			Creator     struct {
				Nickname string `json:"nickname"`
			} `json:"creator"`
			Tracks []struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				Duration int    `json:"duration"`
				Artists  []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name   string `json:"name"`
					PicUrl string `json:"picUrl"`
				} `json:"album"`
			} `json:"tracks"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "Invalid playlist detail JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	songs := make([]ChartSongItem, 0, len(raw.Result.Tracks))
	for _, t := range raw.Result.Tracks {
		singerNames := make([]string, 0, len(t.Artists))
		for _, a := range t.Artists {
			singerNames = append(singerNames, a.Name)
		}
		singer := ""
		if len(singerNames) > 0 {
			singer = singerNames[0]
			for i := 1; i < len(singerNames); i++ {
				singer += ", " + singerNames[i]
			}
		}
		if singer == "" {
			singer = "群星"
		}

		id := strconv.FormatInt(t.ID, 10)
		durSec := t.Duration / 1000
		if durSec <= 0 {
			durSec = 180
		}

		cover := t.Album.PicUrl
		if cover == "" {
			cover = raw.Result.CoverImgUrl
		}

		songs = append(songs, ChartSongItem{
			ID:       "wy_" + id,
			Name:     t.Name,
			Singer:   singer,
			Album:    t.Album.Name,
			Duration: durSec,
			Interval: durSec,
			Cover:    cover,
			Source:   "wy",
			Songmid:  id,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":          raw.Result.ID,
			"name":        raw.Result.Name,
			"cover":       raw.Result.CoverImgUrl,
			"description": raw.Result.Description,
			"play_count":  raw.Result.PlayCount,
			"track_count": len(songs),
			"creator":     raw.Result.Creator.Nickname,
			"songs":       songs,
		},
	})
}
