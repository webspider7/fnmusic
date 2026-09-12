package updater

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 当前应用版本号
const CurrentVersion = "1.2.7"

const (
	RepoOwner          = "webspider7"
	RepoName           = "fnmusic"
	ReleaseApiURL      = "https://api.github.com/repos/webspider7/fnmusic/releases/latest"
	GitHubReleasePage  = "https://github.com/webspider7/fnmusic/releases"
	DefaultFpkDownload = "https://github.com/webspider7/fnmusic/releases/latest/download/fn-lx-player.fpk"
)

type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	HtmlURL     string        `json:"html_url"`
	Assets      []GitHubAsset `json:"assets"`
}

type UpdateCheckResult struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	Title          string `json:"title"`
	Changelog      string `json:"changelog"`
	PublishedAt    string `json:"published_at"`
	ReleaseURL     string `json:"release_url"`
	DownloadURL    string `json:"download_url"`
	AssetName      string `json:"asset_name"`
	AssetSize      int64  `json:"asset_size"`
}

type UpdaterManager struct {
	httpClient *http.Client
	cacheMutex sync.RWMutex
	cachedRes  *UpdateCheckResult
	cacheTime  time.Time
}

func NewUpdaterManager() *UpdaterManager {
	return &UpdaterManager{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CompareVersions 比较两个语义化版本号 (例如 "1.2.6" 和 "1.2.7")
// 如果 v1 < v2 返回 -1；v1 == v2 返回 0；v1 > v2 返回 1
func CompareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(strings.TrimSpace(v1), "v")
	v2 = strings.TrimPrefix(strings.TrimSpace(v2), "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		num1 := 0
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}
		num2 := 0
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}
	return 0
}

func (u *UpdaterManager) CheckUpdate(force bool) (*UpdateCheckResult, error) {
	u.cacheMutex.RLock()
	// 5分钟缓存，防止频繁请求命中 GitHub 60次/小时 API 限制
	if !force && u.cachedRes != nil && time.Since(u.cacheTime) < 5*time.Minute {
		res := *u.cachedRes
		u.cacheMutex.RUnlock()
		return &res, nil
	}
	u.cacheMutex.RUnlock()

	req, err := http.NewRequest(http.MethodGet, ReleaseApiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "AuroraMusic-Updater/1.0 (+https://github.com/webspider7/fnmusic)")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch github releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to decode github release: %w", err)
	}

	rawTag := strings.TrimSpace(rel.TagName)
	cleanRemoteVer := strings.TrimPrefix(rawTag, "v")

	// 查找 fpk 资产包
	var downloadURL string
	var assetName string
	var assetSize int64

	for _, asset := range rel.Assets {
		lowerName := strings.ToLower(asset.Name)
		if strings.HasSuffix(lowerName, ".fpk") {
			downloadURL = asset.BrowserDownloadURL
			assetName = asset.Name
			assetSize = asset.Size
			break
		}
	}

	// 兜底下载链接
	if downloadURL == "" {
		if rawTag != "" {
			downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/fn-lx-player.fpk", RepoOwner, RepoName, rawTag)
		} else {
			downloadURL = DefaultFpkDownload
		}
		assetName = "fn-lx-player.fpk"
	}

	releaseURL := rel.HtmlURL
	if releaseURL == "" {
		releaseURL = GitHubReleasePage
	}

	hasUpdate := CompareVersions(CurrentVersion, cleanRemoteVer) < 0

	result := &UpdateCheckResult{
		CurrentVersion: CurrentVersion,
		LatestVersion:  cleanRemoteVer,
		HasUpdate:      hasUpdate,
		Title:          rel.Name,
		Changelog:      rel.Body,
		PublishedAt:    rel.PublishedAt,
		ReleaseURL:     releaseURL,
		DownloadURL:    downloadURL,
		AssetName:      assetName,
		AssetSize:      assetSize,
	}

	u.cacheMutex.Lock()
	u.cachedRes = result
	u.cacheTime = time.Now()
	u.cacheMutex.Unlock()

	return result, nil
}

func (u *UpdaterManager) HandleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	force := r.URL.Query().Get("force") == "1" || r.URL.Query().Get("force") == "true"

	res, err := u.CheckUpdate(force)
	if err != nil {
		log.Printf("[UPDATER] Check update failed: %v", err)
		// 即使 GitHub 接口网络不稳定，也返回当前版本与 fallback 说明，避免前端抛红
		fallback := &UpdateCheckResult{
			CurrentVersion: CurrentVersion,
			LatestVersion:  CurrentVersion,
			HasUpdate:      false,
			Title:          "当前版本: v" + CurrentVersion,
			Changelog:      "无法连接到 GitHub 检查更新，请确认网络连接或直接访问 GitHub Release 页面查看。",
			PublishedAt:    "",
			ReleaseURL:     GitHubReleasePage,
			DownloadURL:    DefaultFpkDownload,
			AssetName:      "fn-lx-player.fpk",
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "github_network_warning: " + err.Error(),
			"data":    fallback,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"data":    res,
	})
}
