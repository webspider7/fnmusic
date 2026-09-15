package search

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UnifiedSong struct {
	ID        string   `json:"id"`
	Songmid   string   `json:"songmid"`
	Hash      string   `json:"hash"`
	Name      string   `json:"name"`
	Singer    string   `json:"singer"`
	Album     string   `json:"album"`
	Cover     string   `json:"cover"`
	Source    string   `json:"source"` // "wy", "tx", "kg", "kw", "mg"
	Duration  int      `json:"duration"`
	Interval  int      `json:"interval"`
	Quality   string   `json:"quality"`
	Qualitys  []string `json:"qualitys"`
	RawSource string   `json:"raw_source"`
}

var httpClient = &http.Client{
	Timeout: 8 * time.Second,
}

func isNoisySearchItem(keyword, title, singer, album string, durationSec int) bool {
	kwLower := strings.ToLower(strings.TrimSpace(keyword))
	tLower := strings.ToLower(strings.TrimSpace(title))
	sLower := strings.ToLower(strings.TrimSpace(singer))
	aLower := strings.ToLower(strings.TrimSpace(album))

	if durationSec > 600 || (durationSec > 0 && durationSec < 50 && !strings.Contains(kwLower, "铃声") && !strings.Contains(kwLower, "片段")) {
		return true
	}

	noiseKeywords := []string{
		"伴奏", "ktv", "cover", "翻唱", "片段", "铃声", "剪辑", "纯音乐",
		"喊麦", "慢摇", "串烧", "变速", "降调", "升调", "高潮版", "副歌",
		"dj", "remix", "伴唱", "男声版", "女声版", "烟嗓", "热播", "抖音", "快手",
		"加长版", "慢速", "变调", "弹唱", "电音版", "电音", "dj版", "remix版", "深情版", "dj串烧",
	}
	for _, nk := range noiseKeywords {
		if !strings.Contains(kwLower, nk) {
			if strings.Contains(tLower, nk) || strings.Contains(sLower, nk) {
				return true
			}
		}
	}

	singerNoise := []string{
		"电台", "故事会", "讲故事", "音乐盒", "放映室", "恋人", "解说", "广播", "有声",
	}
	for _, sn := range singerNoise {
		if !strings.Contains(kwLower, sn) && strings.Contains(sLower, sn) {
			return true
		}
	}

	albumNoise := []string{
		"翻唱", "cover", "伴奏", "轻音乐", "钢琴", "吉他", "小提琴", "古筝",
		"纯音乐", "睡眠", "助眠", "减压", "冥想", "胎教", "瑜伽", "白噪音",
		"广播剧", "有声书", "评书", "相声", "小说", "讲故事", "故事", "解说", "睡眠曲",
		"for流浪", "抖音", "快手", "深情版", "经典好歌", "流行歌曲", "大全集", "合集", "合辑",
		"畅听", "网络流行", "音乐驿站", "精选集", "纪录片", "大型纪录片", "麦克阿瑟", "短剧",
	}
	for _, an := range albumNoise {
		if !strings.Contains(kwLower, an) && strings.Contains(aLower, an) {
			return true
		}
	}

	return false
}

type scoredSong struct {
	song  UnifiedSong
	score int
}

func filterAndRankSongs(keyword string, songs []UnifiedSong) []UnifiedSong {
	kwLower := strings.ToLower(strings.TrimSpace(keyword))
	kwTokens := strings.Fields(kwLower)

	type pairKey struct {
		title  string
		singer string
	}
	consensusMap := make(map[pairKey]int)
	for _, s := range songs {
		t := strings.ToLower(strings.TrimSpace(s.Name))
		if idx := strings.Index(t, "("); idx > 0 {
			t = strings.TrimSpace(t[:idx])
		}
		if idx := strings.Index(t, "（"); idx > 0 {
			t = strings.TrimSpace(t[:idx])
		}
		sName := strings.ToLower(strings.TrimSpace(s.Singer))
		sName = strings.ReplaceAll(sName, "g.e.m.", "")
		sName = strings.TrimSpace(sName)
		k := pairKey{title: t, singer: sName}
		consensusMap[k]++
	}

	scored := make([]scoredSong, 0, len(songs))
	noisy := make([]UnifiedSong, 0)

	for idx, s := range songs {
		if isNoisySearchItem(kwLower, s.Name, s.Singer, s.Album, s.Duration) {
			noisy = append(noisy, s)
			continue
		}

		score := 0
		sNameLower := strings.ToLower(strings.TrimSpace(s.Name))
		sSingerLower := strings.ToLower(strings.TrimSpace(s.Singer))
		sAlbumLower := strings.ToLower(strings.TrimSpace(s.Album))

		normTitle := sNameLower
		if i := strings.Index(normTitle, "("); i > 0 {
			normTitle = strings.TrimSpace(normTitle[:i])
		}
		if i := strings.Index(normTitle, "（"); i > 0 {
			normTitle = strings.TrimSpace(normTitle[:i])
		}

		// 1. Multi-token or Single-token matching
		if len(kwTokens) > 1 {
			titleMatched := false
			singerMatched := false
			for _, tok := range kwTokens {
				if normTitle == tok {
					score += 3000
					titleMatched = true
				} else if strings.Contains(normTitle, tok) {
					score += 1500
					titleMatched = true
				}
				if strings.Contains(sSingerLower, tok) {
					score += 3000
					singerMatched = true
				}
			}
			if titleMatched && singerMatched {
				score += 6000 // Golden combo! Both title and singer matched!
			} else if !titleMatched {
				score -= 4000 // Only singer matched but wrong song!
			}
		} else {
			if sNameLower == kwLower {
				score += 3000
			} else if normTitle == kwLower {
				score += 2400
			} else if strings.HasPrefix(sNameLower, kwLower) {
				score += 1000
			} else if strings.Contains(sNameLower, kwLower) {
				score += 400
			} else {
				score -= 2000
			}

			if sSingerLower == kwLower {
				score += 2500
			}
		}

		// 2. Platform authoritative ranking bonus (KuGou & QQ rank popular originals top)
		rawRank := idx % 16
		if s.Source == "kg" || s.Source == "tx" {
			if rawRank == 0 {
				score += 1000
			} else if rawRank == 1 {
				score += 700
			} else if rawRank == 2 {
				score += 450
			}
		} else {
			if rawRank == 0 {
				score += 400
			}
		}

		// 3. Cross-platform consensus bonus
		normSinger := strings.ReplaceAll(sSingerLower, "g.e.m.", "")
		normSinger = strings.TrimSpace(normSinger)
		k := pairKey{title: normTitle, singer: normSinger}
		if c := consensusMap[k]; c >= 2 {
			score += 1500 * c
		}

		// 4. Studio Album Authority
		if sAlbumLower != "" {
			if sAlbumLower == normTitle {
				// Eponymous title track album (e.g. 七里香 in album 七里香)
				score += 600
			} else if !strings.Contains(sAlbumLower, "（") && !strings.Contains(sAlbumLower, "(") {
				score += 400
			}
		}

		// 5. Normal Duration sweet spot (170s - 340s)
		if s.Duration >= 170 && s.Duration <= 340 {
			score += 250
		} else if s.Duration > 0 && s.Duration < 120 {
			score -= 500
		}

		// 6. Quality bonus
		if strings.ToUpper(s.Quality) == "FLAC" {
			score += 150
		}

		scored = append(scored, scoredSong{song: s, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	clean := make([]UnifiedSong, len(scored))
	for i, sc := range scored {
		clean[i] = sc.song
	}

	if len(clean) >= 6 {
		return clean
	}
	return append(clean, noisy...)
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	q := r.URL.Query()
	keyword := strings.TrimSpace(q.Get("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(q.Get("keyword"))
	}
	if keyword == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"data":    map[string]interface{}{"list": []UnifiedSong{}, "total": 0},
			"list":    []UnifiedSong{},
			"total":   0,
			"message": "empty keyword",
		})
		return
	}

	platform := strings.ToLower(q.Get("source"))
	if platform == "" {
		platform = strings.ToLower(q.Get("platform"))
	}
	if platform == "" {
		platform = "all"
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}

	results := make([]UnifiedSong, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex

	appendResults := func(songs []UnifiedSong) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, songs...)
	}

	switch platform {
	case "wy":
		results = filterAndRankSongs(keyword, searchNetEase(keyword, page, pageSize*2))
	case "tx":
		results = filterAndRankSongs(keyword, searchQQ(keyword, page, pageSize*2))
	case "kg":
		results = filterAndRankSongs(keyword, searchKuGou(keyword, page, pageSize*2))
	case "kw":
		results = filterAndRankSongs(keyword, searchKuWo(keyword, page, pageSize*2))
	default: // "all"
		wg.Add(4)
		go func() {
			defer wg.Done()
			appendResults(searchNetEase(keyword, page, 16))
		}()
		go func() {
			defer wg.Done()
			appendResults(searchQQ(keyword, page, 16))
		}()
		go func() {
			defer wg.Done()
			appendResults(searchKuGou(keyword, page, 16))
		}()
		go func() {
			defer wg.Done()
			appendResults(searchKuWo(keyword, page, 16))
		}()
		wg.Wait()
		results = filterAndRankSongs(keyword, results)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":     200,
		"message":  "ok",
		"keyword":  keyword,
		"platform": platform,
		"data": map[string]interface{}{
			"list":     results,
			"total":    len(results),
			"keyword":  keyword,
			"platform": platform,
		},
		"list":  results,
		"total": len(results),
	})
}

func HandleLyric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := r.URL.Query()
	source := strings.ToLower(q.Get("source"))
	songmid := strings.TrimSpace(q.Get("songmid"))
	if songmid == "" {
		songmid = strings.TrimSpace(q.Get("id"))
	}
	title := strings.TrimSpace(q.Get("title"))
	if title == "" {
		title = strings.TrimSpace(q.Get("name"))
	}
	singer := strings.TrimSpace(q.Get("singer"))
	if singer == "" {
		singer = strings.TrimSpace(q.Get("artist"))
	}

	durSec, _ := strconv.Atoi(q.Get("duration"))
	if durSec <= 0 {
		durSec, _ = strconv.Atoi(q.Get("interval"))
	}
	hash := strings.TrimSpace(q.Get("hash"))

	lrc, tlrc := fetchLyric(source, songmid, title, singer, durSec, hash)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]string{
			"lyric":  lrc,
			"tlyric": tlrc,
		},
		"lyric": lrc,
	})
}

func normalizeLyricTitle(s string) string {
	s = strings.ToLower(cleanTitle(s))
	for _, pair := range [][]string{{"(", ")"}, {"（", "）"}, {"[", "]"}, {"【", "】"}} {
		for {
			start := strings.Index(s, pair[0])
			if start == -1 {
				break
			}
			end := strings.Index(s[start:], pair[1])
			if end == -1 {
				break
			}
			s = s[:start] + s[start+end+len(pair[1]):]
		}
	}
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return strings.TrimSpace(s)
}

func titleMatches(t1, t2 string) bool {
	rawTarget := strings.ToLower(cleanTitle(t1))
	rawCand := strings.ToLower(cleanTitle(t2))

	noisyKeywords := []string{"伴奏", "ktv", "cover", "翻唱", "片段", "铃声", "剪辑", "纯音乐", "激情", "弹唱"}
	for _, kw := range noisyKeywords {
		if !strings.Contains(rawTarget, kw) && strings.Contains(rawCand, kw) {
			return false
		}
	}

	n1 := normalizeLyricTitle(t1)
	n2 := normalizeLyricTitle(t2)
	if n1 == "" || n2 == "" {
		return true
	}
	return n1 == n2 || strings.Contains(n1, n2) || strings.Contains(n2, n1)
}

func singerMatches(s1, s2 string) bool {
	s1 = strings.ToLower(cleanTitle(s1))
	s2 = strings.ToLower(cleanTitle(s2))
	if s1 == "" || s2 == "" {
		return true
	}
	s1 = strings.ReplaceAll(s1, " ", "")
	s2 = strings.ReplaceAll(s2, " ", "")
	return strings.Contains(s1, s2) || strings.Contains(s2, s1)
}

func fetchNetEaseLyricByID(songmid string) (string, string) {
	url := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%s&lv=1&kv=1&tv=-1", songmid)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://music.163.com/")
	resp, err := httpClient.Do(req)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		var res struct {
			Lrc struct {
				Lyric string `json:"lyric"`
			} `json:"lrc"`
			Tlyric struct {
				Lyric string `json:"lyric"`
			} `json:"tlyric"`
		}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &res)
		if res.Lrc.Lyric != "" {
			return res.Lrc.Lyric, res.Tlyric.Lyric
		}
	}
	return "", ""
}

func fetchNetEaseLyricBySearch(query, targetName, targetSinger string, durationSec int) (string, string) {
	apiURL := fmt.Sprintf("https://music.163.com/api/cloudsearch/pc?s=%s&type=1&offset=0&limit=5", url.QueryEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://music.163.com/")
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", ""
	}
	defer resp.Body.Close()

	var raw struct {
		Result struct {
			Songs []struct {
				ID      int64  `json:"id"`
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"ar"`
				Duration int `json:"dt"`
			} `json:"songs"`
		} `json:"result"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &raw)

	for _, s := range raw.Result.Songs {
		if isNoisySearchItem("", s.Name, "", "", 0) {
			continue
		}
		if !titleMatches(targetName, s.Name) {
			continue
		}
		artistNames := make([]string, 0)
		for _, a := range s.Artists {
			artistNames = append(artistNames, a.Name)
		}
		artistStr := strings.Join(artistNames, ", ")
		if targetSinger != "" && !singerMatches(targetSinger, artistStr) {
			continue
		}
		// If duration is provided, check tolerance
		if durationSec > 0 && s.Duration > 0 {
			diff := int(math.Abs(float64(s.Duration/1000 - durationSec)))
			if diff > 10 && durationSec > 60 {
				continue
			}
		}
		idStr := strconv.FormatInt(s.ID, 10)
		if lrc, tlrc := fetchNetEaseLyricByID(idStr); lrc != "" {
			return lrc, tlrc
		}
	}
	return "", ""
}

func fetchKuGouLyricStrict(searchKeyword, targetName, targetSinger string, durationSec int, hash string) string {
	durationMs := durationSec * 1000
	searchURL := fmt.Sprintf("http://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword=%s&duration=%d&hash=%s",
		url.QueryEscape(searchKeyword), durationMs, hash)
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var kgRes struct {
		Candidates []struct {
			ID        string      `json:"id"`
			AccessKey string      `json:"accesskey"`
			Duration  int         `json:"duration"`
			Song      string      `json:"song"`
			Singer    string      `json:"singer"`
			Adjust    interface{} `json:"adjust"`
		} `json:"candidates"`
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &kgRes)
	if len(kgRes.Candidates) == 0 {
		return ""
	}

	bestIdx := -1
	minDiff := 999999999

	for idx, cand := range kgRes.Candidates {
		if isNoisySearchItem("", cand.Song, cand.Singer, "", 0) {
			continue
		}
		if !titleMatches(targetName, cand.Song) {
			continue
		}
		if targetSinger != "" && !singerMatches(targetSinger, cand.Singer) {
			continue
		}

		if durationMs > 0 {
			diff := int(math.Abs(float64(cand.Duration - durationMs)))
			if cand.Duration > 50000 && diff < minDiff {
				minDiff = diff
				bestIdx = idx
			}
		} else if cand.Duration > 50000 {
			bestIdx = idx
			break
		}
	}

	if bestIdx == -1 {
		for idx, cand := range kgRes.Candidates {
			if !isNoisySearchItem("", cand.Song, cand.Singer, "", 0) && cand.Duration > 50000 {
				bestIdx = idx
				break
			}
		}
	}

	if bestIdx == -1 {
		return ""
	}

	c := kgRes.Candidates[bestIdx]
	dlURL := fmt.Sprintf("http://lyrics.kugou.com/download?ver=1&client=pc&id=%s&accesskey=%s&fmt=lrc&charset=utf8",
		c.ID, c.AccessKey)
	req2, _ := http.NewRequest("GET", dlURL, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0")
	resp2, err2 := httpClient.Do(req2)
	if err2 == nil && resp2.StatusCode == 200 {
		defer resp2.Body.Close()
		var dlRes struct {
			Content string `json:"content"`
		}
		body2, _ := io.ReadAll(resp2.Body)
		_ = json.Unmarshal(body2, &dlRes)
		if dlRes.Content != "" {
			decoded, decErr := base64.StdEncoding.DecodeString(dlRes.Content)
			if decErr == nil && len(decoded) > 0 {
				lrcStr := string(decoded)
				var adjMs int
				switch v := c.Adjust.(type) {
				case float64:
					adjMs = int(v)
				case string:
					adjMs, _ = strconv.Atoi(v)
				case int:
					adjMs = v
				}
				if adjMs != 0 && !strings.Contains(lrcStr, "[offset:") {
					lrcStr = fmt.Sprintf("[offset:%d]\n%s", adjMs, lrcStr)
				}
				return lrcStr
			}
		}
	}
	return ""
}

func fetchLyric(source, songmid, title, singer string, durationSec int, hash string) (string, string) {
	// 1. If NetEase ID available
	if (source == "wy" || strings.HasPrefix(source, "wy")) && songmid != "" {
		if lrc, tlrc := fetchNetEaseLyricByID(songmid); lrc != "" {
			return lrc, tlrc
		}
	}

	cleanT := cleanTitle(title)
	cleanS := cleanTitle(singer)

	// 2. High-precision Official NetEase CloudSearch (Works universally for QQ, KuGou, KuWo)
	if cleanT != "" {
		searchQ := cleanT
		if cleanS != "" {
			searchQ = cleanS + " " + cleanT
		}
		if lrc, tlrc := fetchNetEaseLyricBySearch(searchQ, cleanT, cleanS, durationSec); lrc != "" {
			return lrc, tlrc
		}
	}

	// 3. Universal Lyric Search via KuGou Lyric Engine (with strict candidate validation)
	if cleanT != "" {
		searchKeyword := cleanT
		if cleanS != "" {
			searchKeyword = cleanS + " - " + cleanT
		}
		if lrc := fetchKuGouLyricStrict(searchKeyword, cleanT, cleanS, durationSec, hash); lrc != "" {
			return lrc, ""
		}
	}

	return "", ""
}

func determineHighestQuality(qualitys []string) string {
	for _, q := range qualitys {
		if q == "flac24bit" || q == "hires" {
			return "Hi-Res"
		}
	}
	for _, q := range qualitys {
		if q == "flac" {
			return "FLAC"
		}
	}
	for _, q := range qualitys {
		if q == "320k" {
			return "320K"
		}
	}
	for _, q := range qualitys {
		if q == "192k" {
			return "192K"
		}
	}
	return "128K"
}

// 1. NetEase (uses cloudsearch for HD cover al.picUrl and real quality presence)
func searchNetEase(keyword string, page, limit int) []UnifiedSong {
	apiURL := fmt.Sprintf("https://music.163.com/api/cloudsearch/pc?s=%s&type=1&offset=%d&limit=%d",
		url.QueryEscape(keyword), (page-1)*limit, limit)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://music.163.com/")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Result struct {
			Songs []struct {
				ID      int64  `json:"id"`
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"ar"`
				Album struct {
					Name   string `json:"name"`
					PicURL string `json:"picUrl"`
				} `json:"al"`
				Duration  int `json:"dt"`
				Privilege struct {
					Maxbr int `json:"maxbr"`
				} `json:"privilege"`
				H  *struct{ Br int `json:"br"` } `json:"h"`
				M  *struct{ Br int `json:"br"` } `json:"m"`
				L  *struct{ Br int `json:"br"` } `json:"l"`
				Sq *struct{ Br int `json:"br"` } `json:"sq"`
				Hr *struct{ Br int `json:"br"` } `json:"hr"`
			} `json:"songs"`
		} `json:"result"`
	}

	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &raw)

	list := make([]UnifiedSong, 0, len(raw.Result.Songs))
	for _, s := range raw.Result.Songs {
		artists := make([]string, 0)
		for _, a := range s.Artists {
			artists = append(artists, a.Name)
		}
		durationSec := s.Duration / 1000
		cover := s.Album.PicURL
		if cover != "" && strings.HasPrefix(cover, "http://") {
			cover = strings.Replace(cover, "http://", "https://", 1)
		}

		qualitys := make([]string, 0, 5)
		if s.L != nil || s.Privilege.Maxbr >= 128000 {
			qualitys = append(qualitys, "128k")
		}
		if s.M != nil || s.Privilege.Maxbr >= 192000 {
			qualitys = append(qualitys, "192k")
		}
		if s.H != nil || s.Privilege.Maxbr >= 320000 {
			qualitys = append(qualitys, "320k")
		}
		if s.Sq != nil || s.Privilege.Maxbr >= 999000 {
			qualitys = append(qualitys, "flac")
		}
		if s.Hr != nil {
			qualitys = append(qualitys, "flac24bit")
		}
		if len(qualitys) == 0 {
			qualitys = []string{"128k"}
		}

		list = append(list, UnifiedSong{
			ID:        fmt.Sprintf("wy_%d", s.ID),
			Songmid:   fmt.Sprintf("%d", s.ID),
			Name:      s.Name,
			Singer:    strings.Join(artists, ", "),
			Album:     s.Album.Name,
			Cover:     cover,
			Source:    "wy",
			Duration:  durationSec,
			Interval:  durationSec,
			Quality:   determineHighestQuality(qualitys),
			Qualitys:  qualitys,
			RawSource: "wy",
		})
	}
	return list
}

// 2. QQ Music (HD album cover via Albummid and real quality presence)
func searchQQ(keyword string, page, limit int) []UnifiedSong {
	apiURL := fmt.Sprintf("https://c.y.qq.com/soso/fcgi-bin/client_search_cp?p=%d&n=%d&w=%s&format=json",
		page, limit, url.QueryEscape(keyword))

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://y.qq.com/")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Song struct {
				List []struct {
					Songmid   string `json:"songmid"`
					Songname  string `json:"songname"`
					Singer    []struct {
						Name string `json:"name"`
					} `json:"singer"`
					Albumname string `json:"albumname"`
					Albummid  string `json:"albummid"`
					Interval  int    `json:"interval"`
					Size128   int64  `json:"size128"`
					Size320   int64  `json:"size320"`
					SizeFlac  int64  `json:"sizeflac"`
					SizeHires int64  `json:"sizehires"`
				} `json:"list"`
			} `json:"song"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &raw)

	list := make([]UnifiedSong, 0, len(raw.Data.Song.List))
	for _, s := range raw.Data.Song.List {
		singers := make([]string, 0)
		for _, a := range s.Singer {
			singers = append(singers, a.Name)
		}
		cover := ""
		if s.Albummid != "" {
			cover = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", s.Albummid)
		}

		qualitys := make([]string, 0, 4)
		if s.Size128 > 0 {
			qualitys = append(qualitys, "128k")
		}
		if s.Size320 > 0 {
			qualitys = append(qualitys, "320k")
		}
		if s.SizeFlac > 0 {
			qualitys = append(qualitys, "flac")
		}
		if s.SizeHires > 0 {
			qualitys = append(qualitys, "flac24bit")
		}
		if len(qualitys) == 0 {
			qualitys = []string{"128k"}
		}

		list = append(list, UnifiedSong{
			ID:        "tx_" + s.Songmid,
			Songmid:   s.Songmid,
			Name:      s.Songname,
			Singer:    strings.Join(singers, ", "),
			Album:     s.Albumname,
			Cover:     cover,
			Source:    "tx",
			Duration:  s.Interval,
			Interval:  s.Interval,
			Quality:   determineHighestQuality(qualitys),
			Qualitys:  qualitys,
			RawSource: "tx",
		})
	}
	return list
}

// 3. KuGou (HD album cover via Image field and real quality presence)
func searchKuGou(keyword string, page, limit int) []UnifiedSong {
	apiURL := fmt.Sprintf("https://songsearch.kugou.com/song_search_v2?keyword=%s&page=%d&pagesize=%d&platform=WebFilter",
		url.QueryEscape(keyword), page, limit)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Data struct {
			Lists []struct {
				FileHash    string `json:"FileHash"`
				FileSize    int64  `json:"FileSize"`
				HQFileHash  string `json:"HQFileHash"`
				HQFileSize  int64  `json:"HQFileSize"`
				SQFileHash  string `json:"SQFileHash"`
				SQFileSize  int64  `json:"SQFileSize"`
				ResFileHash string `json:"ResFileHash"`
				ResFileSize int64  `json:"ResFileSize"`
				SongName    string `json:"SongName"`
				SingerName  string `json:"SingerName"`
				AlbumName   string `json:"AlbumName"`
				Duration    int    `json:"Duration"`
				Image       string `json:"Image"`
			} `json:"lists"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &raw)

	list := make([]UnifiedSong, 0, len(raw.Data.Lists))
	for _, s := range raw.Data.Lists {
		name := strings.ReplaceAll(strings.ReplaceAll(s.SongName, "<em>", ""), "</em>", "")
		singer := strings.ReplaceAll(strings.ReplaceAll(s.SingerName, "<em>", ""), "</em>", "")
		cover := ""
		if s.Image != "" {
			cover = strings.ReplaceAll(s.Image, "{size}", "400")
			if strings.HasPrefix(cover, "http://") {
				cover = strings.Replace(cover, "http://", "https://", 1)
			}
		}

		qualitys := make([]string, 0, 4)
		if s.FileHash != "" && s.FileSize > 0 {
			qualitys = append(qualitys, "128k")
		}
		if s.HQFileHash != "" && s.HQFileSize > 0 {
			qualitys = append(qualitys, "320k")
		}
		if s.SQFileHash != "" && s.SQFileSize > 0 {
			qualitys = append(qualitys, "flac")
		}
		if s.ResFileHash != "" && s.ResFileSize > 0 {
			qualitys = append(qualitys, "flac24bit")
		}
		if len(qualitys) == 0 {
			qualitys = []string{"128k"}
		}

		list = append(list, UnifiedSong{
			ID:        "kg_" + s.FileHash,
			Songmid:   s.FileHash,
			Hash:      s.FileHash,
			Name:      name,
			Singer:    singer,
			Album:     s.AlbumName,
			Cover:     cover,
			Source:    "kg",
			Duration:  s.Duration,
			Interval:  s.Duration,
			Quality:   determineHighestQuality(qualitys),
			Qualitys:  qualitys,
			RawSource: "kg",
		})
	}
	return list
}

// 4. KuWo (HD album cover via web_albumpic_short and real quality presence)
func searchKuWo(keyword string, page, limit int) []UnifiedSong {
	apiURL := fmt.Sprintf("https://search.kuwo.cn/r.s?all=%s&ft=music&itemset=web_2013&client=kt&pn=%d&rn=%d&rformat=json&encoding=utf8",
		url.QueryEscape(keyword), page-1, limit)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Abslist []struct {
			MUSICRID          string      `json:"MUSICRID"`
			SONGNAME          string      `json:"SONGNAME"`
			ARTIST            string      `json:"ARTIST"`
			ALBUM             string      `json:"ALBUM"`
			DURATION          interface{} `json:"DURATION"`
			WebAlbumpicShort  string      `json:"web_albumpic_short"`
			WebArtistpicShort string      `json:"web_artistpic_short"`
			N_MINFO           string      `json:"N_MINFO"`
			FORMATS           string      `json:"FORMATS"`
		} `json:"abslist"`
	}

	body, _ := io.ReadAll(resp.Body)
	cleanBody := strings.ReplaceAll(string(body), "'", "\"")
	_ = json.Unmarshal([]byte(cleanBody), &raw)

	list := make([]UnifiedSong, 0, len(raw.Abslist))
	for _, s := range raw.Abslist {
		rid := strings.TrimPrefix(s.MUSICRID, "MUSIC_")
		durSec := 0
		switch v := s.DURATION.(type) {
		case float64:
			durSec = int(v)
		case string:
			durSec, _ = strconv.Atoi(v)
		case int:
			durSec = v
		}

		cover := ""
		if s.WebAlbumpicShort != "" {
			hdPath := strings.Replace(s.WebAlbumpicShort, "120/", "500/", 1)
			cover = "https://img1.kuwo.cn/star/albumcover/" + hdPath
		} else if s.WebArtistpicShort != "" {
			cover = "https://img1.kuwo.cn/star/starheads/" + s.WebArtistpicShort
		}

		qualitys := make([]string, 0, 4)
		formatsUpper := strings.ToUpper(s.FORMATS + " " + s.N_MINFO)
		if strings.Contains(formatsUpper, "128") || strings.Contains(formatsUpper, "LEVEL:H") || strings.Contains(formatsUpper, "MP3128") {
			qualitys = append(qualitys, "128k")
		}
		if strings.Contains(formatsUpper, "320") || strings.Contains(formatsUpper, "LEVEL:P") || strings.Contains(formatsUpper, "MP3H") {
			qualitys = append(qualitys, "320k")
		}
		if strings.Contains(formatsUpper, "FLAC") || strings.Contains(formatsUpper, "LEVEL:ZP") || strings.Contains(formatsUpper, "ALFLAC") || strings.Contains(formatsUpper, "2000") {
			qualitys = append(qualitys, "flac")
		}
		if strings.Contains(formatsUpper, "HIR") || strings.Contains(formatsUpper, "24BIT") {
			qualitys = append(qualitys, "flac24bit")
		}
		if len(qualitys) == 0 {
			qualitys = []string{"128k", "320k"}
		}

		list = append(list, UnifiedSong{
			ID:        "kw_" + rid,
			Songmid:   rid,
			Name:      cleanTitle(s.SONGNAME),
			Singer:    cleanTitle(s.ARTIST),
			Album:     cleanTitle(s.ALBUM),
			Cover:     cover,
			Source:    "kw",
			Duration:  durSec,
			Interval:  durSec,
			Quality:   determineHighestQuality(qualitys),
			Qualitys:  qualitys,
			RawSource: "kw",
		})
	}
	return list
}

func cleanTitle(t string) string {
	t = strings.ReplaceAll(t, "&nbsp;", " ")
	t = strings.ReplaceAll(t, "&amp;", "&")
	t = strings.ReplaceAll(t, "&quot;", "\"")
	t = strings.ReplaceAll(t, "&apos;", "'")
	return strings.TrimSpace(t)
}

// GetSongQualities dynamically retrieves available audio qualities for a given song across sources
func GetSongQualities(source, id, songmid, hash string) []string {
	source = strings.ToLower(strings.TrimSpace(source))
	cleanId := strings.TrimPrefix(id, source+"_")
	cleanMid := strings.TrimPrefix(songmid, source+"_")
	cleanHash := strings.TrimPrefix(hash, source+"_")

	switch source {
	case "wy":
		targetId := cleanId
		if targetId == "" {
			targetId = cleanMid
		}
		if targetId != "" {
			apiURL := fmt.Sprintf("https://music.163.com/api/v3/song/detail?c=[{\"id\":%s}]", targetId)
			req, _ := http.NewRequest("GET", apiURL, nil)
			req.Header.Set("Referer", "https://music.163.com/")
			resp, err := httpClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()
				var det struct {
					Privileges []struct {
						Maxbr int `json:"maxbr"`
					} `json:"privileges"`
					Songs []struct {
						L  *struct{ Br int } `json:"l"`
						M  *struct{ Br int } `json:"m"`
						H  *struct{ Br int } `json:"h"`
						Sq *struct{ Br int } `json:"sq"`
						Hr *struct{ Br int } `json:"hr"`
					} `json:"songs"`
				}
				if json.NewDecoder(resp.Body).Decode(&det) == nil && len(det.Songs) > 0 {
					s := det.Songs[0]
					maxbr := 0
					if len(det.Privileges) > 0 {
						maxbr = det.Privileges[0].Maxbr
					}
					qs := make([]string, 0, 5)
					if s.L != nil || maxbr >= 128000 {
						qs = append(qs, "128k")
					}
					if s.M != nil || maxbr >= 192000 {
						qs = append(qs, "192k")
					}
					if s.H != nil || maxbr >= 320000 {
						qs = append(qs, "320k")
					}
					if s.Sq != nil || maxbr >= 999000 {
						qs = append(qs, "flac")
					}
					if s.Hr != nil {
						qs = append(qs, "flac24bit")
					}
					if len(qs) > 0 {
						return qs
					}
				}
			}
		}
	case "kg":
		targetHash := cleanHash
		if targetHash == "" {
			targetHash = cleanId
		}
		if targetHash != "" {
			// Kugou hash info search
			apiURL := fmt.Sprintf("https://m.kugou.com/app/i/getSongInfo.php?cmd=playInfo&hash=%s", targetHash)
			req, _ := http.NewRequest("GET", apiURL, nil)
			resp, err := httpClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()
				var kgInfo struct {
					FileSize    int64  `json:"fileSize"`
					ExtName     string `json:"extName"`
					HQFileHash  string `json:"hqFileHash"`
					SQFileHash  string `json:"sqFileHash"`
					ResFileHash string `json:"resFileHash"`
				}
				if json.NewDecoder(resp.Body).Decode(&kgInfo) == nil && (kgInfo.HQFileHash != "" || kgInfo.SQFileHash != "") {
					qs := []string{"128k"}
					if kgInfo.HQFileHash != "" {
						qs = append(qs, "320k")
					}
					if kgInfo.SQFileHash != "" {
						qs = append(qs, "flac")
					}
					if kgInfo.ResFileHash != "" {
						qs = append(qs, "flac24bit")
					}
					return qs
				}
			}
		}
		// 官方 API 失效或无有效数据：乐观策略返回完整音质列表，由前端 lx-runtime 实际解析时降级
		return []string{"128k", "320k", "flac"}
	case "kw":
		targetRid := cleanId
		if targetRid == "" {
			targetRid = cleanMid
		}
		if targetRid != "" {
			// Kuwo songinfo
			apiURL := fmt.Sprintf("http://m.kuwo.cn/newh5/singles/songinfoandlrc?musicId=%s", targetRid)
			req, _ := http.NewRequest("GET", apiURL, nil)
			resp, err := httpClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				defer resp.Body.Close()
				var kwRes struct {
					Data struct {
						Formats string `json:"formats"`
					} `json:"data"`
				}
				if json.NewDecoder(resp.Body).Decode(&kwRes) == nil && kwRes.Data.Formats != "" {
					fUpper := strings.ToUpper(kwRes.Data.Formats)
					qs := make([]string, 0, 4)
					if strings.Contains(fUpper, "128") || strings.Contains(fUpper, "MP3128") {
						qs = append(qs, "128k")
					}
					if strings.Contains(fUpper, "320") || strings.Contains(fUpper, "MP3H") {
						qs = append(qs, "320k")
					}
					if strings.Contains(fUpper, "FLAC") || strings.Contains(fUpper, "2000") {
						qs = append(qs, "flac")
					}
					if strings.Contains(fUpper, "HIR") || strings.Contains(fUpper, "24BIT") {
						qs = append(qs, "flac24bit")
					}
					if len(qs) > 0 {
						return qs
					}
				}
			}
		}
		// 官方 API 失效：乐观策略返回完整音质列表，由前端 lx-runtime 实际解析时降级
		return []string{"128k", "320k", "flac"}
	case "tx":
		// 腾讯音乐官方 API 需要鉴权，直接使用乐观策略
		return []string{"128k", "320k", "flac"}
	}

	return []string{"128k", "320k"}
}


// HandleSongQualities handles GET /api/music/qualities
func HandleSongQualities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := r.URL.Query()
	source := strings.ToLower(q.Get("source"))
	id := strings.TrimSpace(q.Get("id"))
	songmid := strings.TrimSpace(q.Get("songmid"))
	hash := strings.TrimSpace(q.Get("hash"))

	qualities := GetSongQualities(source, id, songmid, hash)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":      200,
		"qualities": qualities,
		"highest":   determineHighestQuality(qualities),
	})
}
