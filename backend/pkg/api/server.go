package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"
	"time"

	"fn-lx-player/pkg/charts"
	"fn-lx-player/pkg/config"
	"fn-lx-player/pkg/downloader"
	"fn-lx-player/pkg/nas"
	"fn-lx-player/pkg/proxy"
	"fn-lx-player/pkg/search"
	"fn-lx-player/pkg/sources"
	"fn-lx-player/pkg/updater"
)

func init() {
	_ = mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
	_ = mime.AddExtensionType(".mjs", "text/javascript; charset=utf-8")
	_ = mime.AddExtensionType(".css", "text/css; charset=utf-8")
	_ = mime.AddExtensionType(".html", "text/html; charset=utf-8")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
	_ = mime.AddExtensionType(".png", "image/png")
	_ = mime.AddExtensionType(".json", "application/json; charset=utf-8")
}

type Server struct {
	cfgMgr     *config.ConfigManager
	sourcesMgr *sources.Manager
	chartMgr   *charts.ChartManager
	downloader *downloader.Downloader
	updaterMgr *updater.UpdaterManager
	staticFS   fs.FS
}

func NewServer(cfgMgr *config.ConfigManager, sourcesMgr *sources.Manager, staticFS fs.FS) *Server {
	return &Server{
		cfgMgr:     cfgMgr,
		sourcesMgr: sourcesMgr,
		chartMgr:   charts.NewChartManager(),
		downloader: downloader.NewDownloader(cfgMgr),
		updaterMgr: updater.NewUpdaterManager(),
		staticFS:   staticFS,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// 0. App Info & Auto Update
	mux.HandleFunc("/api/app/check_update", s.updaterMgr.HandleCheckUpdate)
	mux.HandleFunc("/api/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"version": updater.CurrentVersion,
				"name":    "fn-lx-player",
			},
		})
	})

	// 1. Proxy
	mux.HandleFunc("/api/proxy/http", proxy.HandleProxyHTTP)

	// 2. Sources
	mux.HandleFunc("/api/sources", s.sourcesMgr.HandleListSources)
	mux.HandleFunc("/api/sources/script", s.sourcesMgr.HandleGetScript)
	mux.HandleFunc("/api/sources/active", s.sourcesMgr.HandleSetActive)
	mux.HandleFunc("/api/sources/custom", s.sourcesMgr.HandleAddCustom)
	mux.HandleFunc("/api/sources/upload", s.sourcesMgr.HandleAddCustom)
	mux.HandleFunc("/api/sources/import_url", s.sourcesMgr.HandleImportURL)
	mux.HandleFunc("/api/sources/custom/delete", s.sourcesMgr.HandleDeleteCustom)
	mux.HandleFunc("/api/sources/", s.sourcesMgr.HandleSourcesRest)

	// 2.5 Charts & Playlists
	mux.HandleFunc("/api/charts/toplists", s.chartMgr.HandleToplists)
	mux.HandleFunc("/api/charts/detail", s.chartMgr.HandleChartDetail)
	mux.HandleFunc("/api/charts/playlists", s.chartMgr.HandlePlaylists)
	mux.HandleFunc("/api/charts/playlist/detail", s.chartMgr.HandlePlaylistDetail)

	// 2.6 Downloader to NAS
	mux.HandleFunc("/api/download/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.downloader.HandleSetConfig(w, r)
		} else {
			s.downloader.HandleGetConfig(w, r)
		}
	})
	mux.HandleFunc("/api/download/song", s.downloader.HandleDownloadSong)
	mux.HandleFunc("/api/download/batch", s.downloader.HandleDownloadBatch)
	mux.HandleFunc("/api/download/batch/status", s.downloader.HandleBatchStatus)
	mux.HandleFunc("/api/download/check", s.downloader.HandleCheckDownloaded)

	// 3. NAS Local Music
	mux.HandleFunc("/api/nas/folders", nas.ListFolders)
	mux.HandleFunc("/api/nas/directories", nas.ListFolders)
	mux.HandleFunc("/api/nas/browse", nas.BrowseDirectory)
	mux.HandleFunc("/api/nas/songs", nas.ScanSongs)
	mux.HandleFunc("/api/nas/scan", nas.ScanSongs)
	mux.HandleFunc("/api/nas/stream", nas.StreamAudio)
	mux.HandleFunc("/api/nas/cover", nas.ExtractCover)
	mux.HandleFunc("/api/nas/lyric", nas.HandleNasLyric)

	// 4. Search & Direct Audio Resolver
	mux.HandleFunc("/api/search", search.HandleSearch)
	mux.HandleFunc("/api/search/lyric", search.HandleLyric)
	mux.HandleFunc("/api/player/resolve", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":  403,
			"error": "本播放器为纯本地播放器容器，不内置在线音源直链解析。请在前端「音源管理」导入第三方音源脚本。",
		})
	})

	mux.HandleFunc("/api/player/stream", handleAudioStream)

	// 5. Config
	mux.HandleFunc("/api/config", s.handleConfig)

	// 6. Robust Embedded Frontend Static files handler
	if s.staticFS != nil {
		fileServer := http.FileServer(http.FS(s.staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			f, err := s.staticFS.Open(path)
			if err != nil {
				// SPA fallback to index.html
				indexFile, err := s.staticFS.Open("index.html")
				if err != nil {
					http.NotFound(w, r)
					return
				}
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				_, _ = io.Copy(w, indexFile)
				return
			}
			_ = f.Close()

			if path == "index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			} else if strings.HasPrefix(path, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	return corsMiddleware(mux)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch r.Method {
	case http.MethodGet:
		cfg := s.cfgMgr.Get()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "ok",
			"data":    cfg,
			"config":  cfg,
		})
	case http.MethodPost:
		var patch config.AppConfig
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.cfgMgr.Update(patch); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg := s.cfgMgr.Get()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "ok",
			"data":    cfg,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var streamClient = &http.Client{
	Timeout: 30 * time.Second,
}

func handleAudioStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type, Accept")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
	w.Header().Set("Accept-Ranges", "bytes")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	targetURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if targetURL == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}

	outReq, err := http.NewRequest(r.Method, targetURL, nil)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	outReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")

	customReferer := strings.TrimSpace(r.URL.Query().Get("referer"))
	if ref := downloader.GetStreamReferer(targetURL, customReferer); ref != "" {
		outReq.Header.Set("Referer", ref)
	}

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		outReq.Header.Set("Range", rangeHeader)
	}

	resp, err := streamClient.Do(outReq)
	if err != nil {
		http.Error(w, "stream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type, Accept")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
	w.Header().Set("Accept-Ranges", "bytes")

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		w.Header().Set("Content-Range", cr)
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

