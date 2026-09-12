package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"fn-lx-player/pkg/config"
)

var illegalChars = regexp.MustCompile(`[\\/:*?"<>|\r\n\t]`)

func sanitizeFilename(s string) string {
	s = illegalChars.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	if s == "" {
		s = "未知"
	}
	return s
}

type SongPayload struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Singer    string `json:"singer"`
	Album     string `json:"album"`
	Cover     string `json:"cover"`
	Source    string `json:"source"`
	Songmid   string `json:"songmid"`
	URL       string `json:"url"`
	StreamURL string `json:"streamUrl"`
	Referer   string `json:"referer"`
}

// GetStreamReferer determines the appropriate Referer header for music streaming and downloading
func GetStreamReferer(rawURL, customReferer string) string {
	if customReferer != "" {
		return customReferer
	}
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, "qq.com") {
		return "https://y.qq.com/"
	}
	if strings.Contains(lower, "126.net") || strings.Contains(lower, "163.com") {
		return "https://music.163.com/"
	}
	if strings.Contains(lower, "kuwo.cn") {
		return "http://www.kuwo.cn/"
	}
	if strings.Contains(lower, "kugou.com") {
		return "https://www.kugou.com/"
	}
	if strings.Contains(lower, "migu.cn") {
		return "https://music.migu.cn/"
	}
	return ""
}

type BatchTask struct {
	ID          string        `json:"id"`
	Total       int           `json:"total"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	CurrentSong string        `json:"current_song"`
	Status      string        `json:"status"` // "running", "finished"
	Songs       []SongPayload `json:"-"`
	Results     []SongResult  `json:"results"`
}

type SongResult struct {
	SongName string `json:"song_name"`
	Singer   string `json:"singer"`
	Status   string `json:"status"` // "success", "already_exists", "failed"
	Path     string `json:"path"`
	Error    string `json:"error,omitempty"`
}

type Downloader struct {
	cfgMgr     *config.ConfigManager
	httpClient *http.Client
	mu         sync.RWMutex
	batchTasks map[string]*BatchTask
}

func NewDownloader(cfgMgr *config.ConfigManager) *Downloader {
	return &Downloader{
		cfgMgr: cfgMgr,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		batchTasks: make(map[string]*BatchTask),
	}
}

func (d *Downloader) getDownloadDir() string {
	cfg := d.cfgMgr.Get()
	dir := cfg.DownloadDir
	if dir == "" {
		dir = cfg.DefaultNasDir
	}
	if dir == "" {
		dir = "/vol1/Music"
	}
	return filepath.Clean(dir)
}

func (d *Downloader) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	cfg := d.cfgMgr.Get()
	dir := d.getDownloadDir()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":            200,
		"download_dir":    dir,
		"default_nas_dir": cfg.DefaultNasDir,
	})
}

func (d *Downloader) HandleSetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DownloadDir string `json:"download_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	targetDir := strings.TrimSpace(req.DownloadDir)
	if targetDir == "" {
		http.Error(w, "download_dir required", http.StatusBadRequest)
		return
	}

	// Try create dir if not exists
	_ = os.MkdirAll(targetDir, 0755)

	cfg := d.cfgMgr.Get()
	cfg.DownloadDir = targetDir
	if err := d.cfgMgr.Update(cfg); err != nil {
		http.Error(w, "failed to update config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":         200,
		"message":      "ok",
		"download_dir": targetDir,
	})
}

func (d *Downloader) HandleDownloadSong(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var song SongPayload
	if err := json.NewDecoder(r.Body).Decode(&song); err != nil {
		http.Error(w, "invalid song payload", http.StatusBadRequest)
		return
	}

	res, err := d.downloadSingleSong(song)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": err.Error(),
			"result":  res,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    res,
		"result":  res,
	})
}

func (d *Downloader) downloadSingleSong(song SongPayload) (*SongResult, error) {
	destDir := d.getDownloadDir()
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return &SongResult{
			SongName: song.Name,
			Singer:   song.Singer,
			Status:   "failed",
			Error:    "创建下载目录失败: " + err.Error(),
		}, err
	}

	cleanSinger := sanitizeFilename(song.Singer)
	cleanName := sanitizeFilename(song.Name)
	filename := fmt.Sprintf("%s - %s.mp3", cleanSinger, cleanName)
	finalPath := filepath.Join(destDir, filename)

	// Check if already exists and size > 500KB
	if fi, err := os.Stat(finalPath); err == nil && fi.Size() > 500*1024 {
		return &SongResult{
			SongName: song.Name,
			Singer:   song.Singer,
			Status:   "already_exists",
			Path:     filepath.ToSlash(finalPath),
		}, nil
	}

	// Resolve download audio URL
	audioURL := song.URL
	if audioURL == "" || strings.Contains(audioURL, "notice") || strings.Contains(audioURL, "panspace") {
		if song.StreamURL != "" && strings.HasPrefix(song.StreamURL, "http") {
			audioURL = song.StreamURL
		}
	}

	if audioURL == "" || strings.Contains(audioURL, "notice") || strings.Contains(audioURL, "panspace") {
		return &SongResult{
			SongName: song.Name,
			Singer:   song.Singer,
			Status:   "failed",
			Error:    "未提供有效音频流地址，请先通过第三方音源解析后再发起下载",
		}, fmt.Errorf("未提供有效音频流地址")
	}

	// Download to temp file
	tmpPath := finalPath + ".downloading"
	req, err := http.NewRequest("GET", audioURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if ref := GetStreamReferer(audioURL, song.Referer); ref != "" {
		req.Header.Set("Referer", ref)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP 错误: %d", resp.StatusCode)
	}

	out, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	if written < 300*1024 {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("下载文件过小 (可能为提示音频或无效流)")
	}

	// Rename temp to final
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("保存最终文件失败: %w", err)
	}

	// Save cover if available
	if song.Cover != "" && strings.HasPrefix(song.Cover, "http") {
		go d.downloadCover(song.Cover, filepath.Join(destDir, fmt.Sprintf("%s - %s.jpg", cleanSinger, cleanName)))
	}

	return &SongResult{
		SongName: song.Name,
		Singer:   song.Singer,
		Status:   "success",
		Path:     filepath.ToSlash(finalPath),
	}, nil
}

func (d *Downloader) downloadCover(coverURL, targetPath string) {
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return
	}
	req, err := http.NewRequest("GET", coverURL, nil)
	if err != nil {
		return
	}
	resp, err := d.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	f, err := os.Create(targetPath)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = io.Copy(f, resp.Body)
}

func (d *Downloader) HandleCheckDownloaded(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Songs []SongPayload `json:"songs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	destDir := d.getDownloadDir()
	existsMap := make(map[string]bool)

	for _, s := range req.Songs {
		cleanSinger := sanitizeFilename(s.Singer)
		cleanName := sanitizeFilename(s.Name)
		filename := fmt.Sprintf("%s - %s.mp3", cleanSinger, cleanName)
		fullPath := filepath.Join(destDir, filename)
		key := fmt.Sprintf("%s - %s", s.Singer, s.Name)
		if fi, err := os.Stat(fullPath); err == nil && fi.Size() > 300*1024 {
			existsMap[key] = true
			if s.ID != "" {
				existsMap[s.ID] = true
			}
		} else {
			existsMap[key] = false
			if s.ID != "" {
				existsMap[s.ID] = false
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":   200,
		"exists": existsMap,
	})
}

func (d *Downloader) HandleDownloadBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Songs []SongPayload `json:"songs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Songs) == 0 {
		http.Error(w, "empty songs list", http.StatusBadRequest)
		return
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &BatchTask{
		ID:        taskID,
		Total:     len(req.Songs),
		Completed: 0,
		Failed:    0,
		Status:    "running",
		Songs:     req.Songs,
		Results:   make([]SongResult, 0, len(req.Songs)),
	}

	d.mu.Lock()
	d.batchTasks[taskID] = task
	d.mu.Unlock()

	go d.processBatchTask(task)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"task_id": taskID,
		"message": "Batch download started",
		"total":   len(req.Songs),
	})
}

func (d *Downloader) processBatchTask(task *BatchTask) {
	sem := make(chan struct{}, 3) // Concurrency 3
	var wg sync.WaitGroup

	for _, song := range task.Songs {
		sem <- struct{}{}
		wg.Add(1)

		d.mu.Lock()
		task.CurrentSong = fmt.Sprintf("%s - %s", song.Singer, song.Name)
		d.mu.Unlock()

		go func(s SongPayload) {
			defer func() {
				<-sem
				wg.Done()
			}()

			res, err := d.downloadSingleSong(s)
			d.mu.Lock()
			defer d.mu.Unlock()
			if err != nil {
				task.Failed++
				if res != nil {
					task.Results = append(task.Results, *res)
				}
			} else {
				task.Completed++
				if res != nil {
					task.Results = append(task.Results, *res)
				}
			}
		}(song)
	}

	wg.Wait()

	d.mu.Lock()
	task.Status = "finished"
	task.CurrentSong = "全部下载完成"
	d.mu.Unlock()
}

func (d *Downloader) HandleBatchStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "task_id required", http.StatusBadRequest)
		return
	}

	d.mu.RLock()
	task, exists := d.batchTasks[taskID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": task,
	})
}
