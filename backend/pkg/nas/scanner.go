package nas

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type LocalSong struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Duration  int    `json:"duration"` // in seconds
	Interval  int    `json:"interval"`
	Size      int64  `json:"size"`
	Format    string `json:"format"`
	CoverURL  string `json:"cover_url"`
	Cover     string `json:"cover"`
	HasLyric  bool   `json:"has_lyric"`
	UpdatedAt int64  `json:"updated_at"`
}

var audioExts = map[string]bool{
	".mp3":  true,
	".flac": true,
	".wav":  true,
	".ape":  true,
	".m4a":  true,
	".aac":  true,
	".ogg":  true,
	".dsf":  true,
	".dff":  true,
}

type FolderItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	HasSubdirs bool   `json:"has_subdirs"`
	AudioCount int    `json:"audio_count"`
}

type BrowseData struct {
	Current    string       `json:"current"`
	Parent     string       `json:"parent"`
	Volumes    []string     `json:"volumes"`
	Folders    []FolderItem `json:"folders"`
	AudioCount int          `json:"audio_count"`
}

func getAvailableVolumes() []string {
	vols := make([]string, 0)
	for i := 1; i <= 16; i++ {
		p := fmt.Sprintf("/vol%d", i)
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			vols = append(vols, p)
		}
	}
	for _, p := range []string{"/media", "/mnt", "/home"} {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			vols = append(vols, p)
		}
	}
	if len(vols) == 0 {
		vols = []string{"/vol1", "/vol2", "/vol3"}
	}
	return vols
}

func BrowseDirectory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	volumes := getAvailableVolumes()
	rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if rawPath == "" {
		if len(volumes) > 0 {
			rawPath = volumes[0]
		} else {
			rawPath = "/vol1"
		}
	}

	cleanPath := filepath.ToSlash(filepath.Clean(rawPath))
	var parent string
	if cleanPath == "/" || cleanPath == "." || cleanPath == "" {
		cleanPath = "/"
		parent = ""
	} else {
		p := filepath.ToSlash(filepath.Dir(cleanPath))
		if p == cleanPath || p == "." {
			parent = ""
		} else {
			parent = p
		}
	}

	folders := make([]FolderItem, 0)
	audioCount := 0

	entries, err := os.ReadDir(cleanPath)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") || strings.HasPrefix(name, "@") {
				continue
			}
			fullPath := filepath.ToSlash(filepath.Join(cleanPath, name))
			if e.IsDir() {
				hasSub := false
				subAudio := 0
				if subEntries, sErr := os.ReadDir(fullPath); sErr == nil {
					for _, se := range subEntries {
						sName := se.Name()
						if strings.HasPrefix(sName, ".") {
							continue
						}
						if se.IsDir() {
							hasSub = true
						} else {
							ext := strings.ToLower(filepath.Ext(sName))
							if audioExts[ext] {
								subAudio++
							}
						}
					}
				}
				folders = append(folders, FolderItem{
					Name:       name,
					Path:       fullPath,
					HasSubdirs: hasSub,
					AudioCount: subAudio,
				})
			} else {
				ext := strings.ToLower(filepath.Ext(name))
				if audioExts[ext] {
					audioCount++
				}
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"data": BrowseData{
			Current:    cleanPath,
			Parent:     parent,
			Volumes:    volumes,
			Folders:    folders,
			AudioCount: audioCount,
		},
	})
}

func ListFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	presetPaths := []string{
		"/vol1/Music",
		"/vol1/music",
		"/vol1/1000/Music",
		"/vol2/Music",
		"/vol2/music",
		"/vol3/Music",
		"/vol1",
		"/vol2",
		"/vol3",
		"/vol4",
	}

	folders := make([]string, 0)
	for _, p := range presetPaths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			folders = append(folders, p)
		}
	}

	if len(folders) == 0 {
		folders = append(folders, "/vol1/Music", "/vol1/music", "/vol2/Music", "/vol1")
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"data":    folders,
		"folders": folders,
	})
}

func ScanSongs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = "/vol1/Music"
	}

	songs := make([]LocalSong, 0)
	count := 0
	maxCount := 1000

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || count >= maxCount {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if audioExts[ext] {
			count++
			title, artist, album := parseBasicMeta(path, info.Name())
			coverURL := fmt.Sprintf("/api/nas/cover?path=%s", path)
			hasLyric := checkHasLyric(path)
			songs = append(songs, LocalSong{
				ID:        fmt.Sprintf("nas_%d", count),
				Path:      path,
				Filename:  info.Name(),
				Title:     title,
				Artist:    artist,
				Album:     album,
				Size:      info.Size(),
				Format:    strings.TrimPrefix(ext, "."),
				CoverURL:  coverURL,
				Cover:     coverURL,
				HasLyric:  hasLyric,
				UpdatedAt: info.ModTime().Unix(),
			})
		}
		return nil
	})

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "ok",
		"dir":     dir,
		"songs":   songs,
		"total":   len(songs),
		"data": map[string]interface{}{
			"dir":   dir,
			"songs": songs,
			"total": len(songs),
		},
	})
}

func isCleanMeta(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "\ufffd") {
		return false
	}
	printableCount := 0
	for _, r := range s {
		if r >= 32 && r != 127 && r != 0xfffd {
			printableCount++
		}
	}
	return printableCount > 0
}

func parseBasicMeta(filePath, filename string) (title, artist, album string) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(base, " - ")
	if len(parts) >= 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(parts[1])
	} else {
		title = base
		artist = "本地歌手"
	}
	album = "NAS 音乐"

	// Try extracting ID3v2 if mp3
	if strings.ToLower(filepath.Ext(filePath)) == ".mp3" {
		if t, a, al, ok := readID3v2Tags(filePath); ok {
			if isCleanMeta(t) {
				title = t
			}
			if isCleanMeta(a) {
				artist = a
			}
			if isCleanMeta(al) {
				album = al
			}
		}
	}
	return
}

func readID3v2Tags(filePath string) (title, artist, album string, ok bool) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	header := make([]byte, 10)
	if _, err := io.ReadFull(f, header); err != nil {
		return
	}
	if string(header[:3]) != "ID3" {
		return
	}

	tagSize := synchsafeInt(header[6:10])
	if tagSize <= 0 || tagSize > 10*1024*1024 {
		return
	}

	body := make([]byte, tagSize)
	if _, err := io.ReadFull(f, body); err != nil {
		return
	}

	reader := bytes.NewReader(body)
	for reader.Len() > 10 {
		var frameHeader [10]byte
		if _, err := reader.Read(frameHeader[:]); err != nil {
			break
		}
		frameID := string(frameHeader[:4])
		frameSize := int64(binary.BigEndian.Uint32(frameHeader[4:8]))
		if frameSize <= 0 || frameSize > int64(reader.Len()) {
			break
		}

		frameData := make([]byte, frameSize)
		if _, err := reader.Read(frameData); err != nil {
			break
		}

		text := decodeID3Text(frameData)
		switch frameID {
		case "TIT2":
			title = text
		case "TPE1":
			artist = text
		case "TALB":
			album = text
		}
	}
	ok = true
	return
}

func decodeID3Text(data []byte) string {
	if len(data) <= 1 {
		return ""
	}
	encoding := data[0]
	raw := data[1:]

	var decoded string

	switch encoding {
	case 1, 2: // UTF-16 (with or without BOM)
		if len(raw) >= 2 {
			var endian binary.ByteOrder = binary.LittleEndian
			start := 0
			// Check BOM
			if raw[0] == 0xFE && raw[1] == 0xFF {
				endian = binary.BigEndian
				start = 2
			} else if raw[0] == 0xFF && raw[1] == 0xFE {
				endian = binary.LittleEndian
				start = 2
			} else if encoding == 2 {
				endian = binary.BigEndian
			}

			u16s := make([]uint16, 0, (len(raw)-start)/2)
			for i := start; i+1 < len(raw); i += 2 {
				val := endian.Uint16(raw[i : i+2])
				if val == 0 { // null terminator
					break
				}
				u16s = append(u16s, val)
			}
			runes := utf16.Decode(u16s)
			decoded = string(runes)
		}
	case 3: // UTF-8
		decoded = string(raw)
	case 0: // ISO-8859-1 or GBK/UTF-8
		if utf8.Valid(raw) {
			decoded = string(raw)
		}
	default:
		if utf8.Valid(raw) {
			decoded = string(raw)
		}
	}

	decoded = strings.Trim(decoded, "\x00\r\n\t ")
	decoded = strings.ReplaceAll(decoded, "\ufffd", "")
	decoded = strings.ReplaceAll(decoded, "\ufeff", "")
	return strings.TrimSpace(decoded)
}

func synchsafeInt(b []byte) int64 {
	return int64(b[0])<<21 | int64(b[1])<<14 | int64(b[2])<<7 | int64(b[3])
}

func StreamAudio(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(filePath)
	if err != nil || fi.IsDir() {
		http.Error(w, "Audio file not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open audio: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".mp3":
		contentType = "audio/mpeg"
	case ".flac":
		contentType = "audio/flac"
	case ".wav":
		contentType = "audio/wav"
	case ".m4a", ".aac":
		contentType = "audio/mp4"
	case ".ogg":
		contentType = "audio/ogg"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

func ExtractCover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	data, mimeType := extractCoverBytes(filePath)
	if len(data) > 0 {
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
		return
	}

	http.NotFound(w, r)
}

func extractCoverBytes(filePath string) ([]byte, string) {
	// 1. Try audio file tags
	if f, err := os.Open(filePath); err == nil {
		defer f.Close()
		ext := strings.ToLower(filepath.Ext(filePath))

		if ext == ".mp3" {
			header := make([]byte, 10)
			if _, err := io.ReadFull(f, header); err == nil && string(header[:3]) == "ID3" {
				tagSize := synchsafeInt(header[6:10])
				if tagSize > 0 && tagSize <= 25*1024*1024 {
					body := make([]byte, tagSize)
					if _, err := io.ReadFull(f, body); err == nil {
						reader := bytes.NewReader(body)
						for reader.Len() > 10 {
							var frameHeader [10]byte
							if _, err := reader.Read(frameHeader[:]); err != nil {
								break
							}
							frameID := string(frameHeader[:4])
							frameSize := int64(binary.BigEndian.Uint32(frameHeader[4:8]))
							if frameSize <= 0 || frameSize > int64(reader.Len()) {
								break
							}
							frameData := make([]byte, frameSize)
							if _, err := reader.Read(frameData); err != nil {
								break
							}
							if frameID == "APIC" && len(frameData) > 10 {
								jpegIdx := bytes.Index(frameData, []byte{0xFF, 0xD8, 0xFF})
								pngIdx := bytes.Index(frameData, []byte{0x89, 0x50, 0x4E, 0x47})
								if jpegIdx != -1 && (pngIdx == -1 || jpegIdx < pngIdx) {
									return frameData[jpegIdx:], "image/jpeg"
								} else if pngIdx != -1 {
									return frameData[pngIdx:], "image/png"
								}
							}
						}
					}
				}
			}
		} else if ext == ".flac" {
			header := make([]byte, 4)
			if _, err := io.ReadFull(f, header); err == nil && string(header) == "fLaC" {
				buf := make([]byte, 10*1024*1024)
				n, _ := io.ReadFull(f, buf)
				if n > 0 {
					data := buf[:n]
					jpegIdx := bytes.Index(data, []byte{0xFF, 0xD8, 0xFF})
					pngIdx := bytes.Index(data, []byte{0x89, 0x50, 0x4E, 0x47})
					if jpegIdx != -1 && (pngIdx == -1 || jpegIdx < pngIdx) {
						return data[jpegIdx:], "image/jpeg"
					} else if pngIdx != -1 {
						return data[pngIdx:], "image/png"
					}
				}
			}
		}
	}

	// 2. Folder cover fallback
	dir := filepath.Dir(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	candidates := []string{
		filepath.Join(dir, base+".jpg"),
		filepath.Join(dir, base+".jpeg"),
		filepath.Join(dir, base+".png"),
		filepath.Join(dir, "cover.jpg"),
		filepath.Join(dir, "cover.png"),
		filepath.Join(dir, "folder.jpg"),
		filepath.Join(dir, "folder.png"),
		filepath.Join(dir, "front.jpg"),
	}
	for _, cp := range candidates {
		if data, err := os.ReadFile(cp); err == nil && len(data) > 0 {
			if strings.HasSuffix(strings.ToLower(cp), ".png") {
				return data, "image/png"
			}
			return data, "image/jpeg"
		}
	}

	return nil, ""
}

// ── NAS 本地歌词同步扫描与读取 ──

func checkHasLyric(filePath string) bool {
	dir := filepath.Dir(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	candidates := []string{
		filepath.Join(dir, base+".lrc"),
		filepath.Join(dir, base+".LRC"),
		filepath.Join(dir, base+".txt"),
	}
	parts := strings.Split(base, " - ")
	if len(parts) == 2 {
		candidates = append(candidates,
			filepath.Join(dir, strings.TrimSpace(parts[1])+".lrc"),
			filepath.Join(dir, strings.TrimSpace(parts[1])+".LRC"),
		)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return true
		}
	}
	return false
}

func HandleNasLyric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	lrcText, source, err := ExtractLyric(filePath)
	if err != nil || strings.TrimSpace(lrcText) == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    404,
			"message": "Lyric not found",
			"data":    nil,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"lyric":  lrcText,
			"source": source,
		},
	})
}

func ExtractLyric(filePath string) (string, string, error) {
	dir := filepath.Dir(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// 1. 同名或歌曲名 .lrc 文件
	candidates := []string{
		filepath.Join(dir, base+".lrc"),
		filepath.Join(dir, base+".LRC"),
		filepath.Join(dir, base+".txt"),
	}
	parts := strings.Split(base, " - ")
	if len(parts) == 2 {
		candidates = append(candidates,
			filepath.Join(dir, strings.TrimSpace(parts[1])+".lrc"),
			filepath.Join(dir, strings.TrimSpace(parts[1])+".LRC"),
		)
	}

	for _, c := range candidates {
		if b, err := os.ReadFile(c); err == nil && len(b) > 0 {
			text := decodeLrcText(b)
			if strings.TrimSpace(text) != "" {
				return text, "local_file", nil
			}
		}
	}

	// 2. 音频文件内嵌 ID3v2 USLT 歌词标签
	if strings.ToLower(filepath.Ext(filePath)) == ".mp3" {
		if lrc := extractEmbeddedID3Lyric(filePath); lrc != "" {
			return lrc, "embedded_id3", nil
		}
	}

	return "", "", fmt.Errorf("no lyric found")
}

func decodeLrcText(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// UTF-8 BOM
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return string(b[3:])
	}
	// UTF-16 LE BOM
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		u16 := make([]uint16, 0, (len(b)-2)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u16 = append(u16, binary.LittleEndian.Uint16(b[i:i+2]))
		}
		return string(utf16.Decode(u16))
	}
	// UTF-16 BE BOM
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		u16 := make([]uint16, 0, (len(b)-2)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u16 = append(u16, binary.BigEndian.Uint16(b[i:i+2]))
		}
		return string(utf16.Decode(u16))
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return string(b)
}

func extractEmbeddedID3Lyric(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	header := make([]byte, 10)
	if _, err := io.ReadFull(f, header); err != nil || string(header[:3]) != "ID3" {
		return ""
	}

	tagSize := synchsafeInt(header[6:10])
	if tagSize <= 0 || tagSize > 25*1024*1024 {
		return ""
	}

	body := make([]byte, tagSize)
	if _, err := io.ReadFull(f, body); err != nil {
		return ""
	}

	reader := bytes.NewReader(body)
	for reader.Len() > 10 {
		var frameHeader [10]byte
		if _, err := reader.Read(frameHeader[:]); err != nil {
			break
		}
		frameID := string(frameHeader[:4])
		frameSize := int64(binary.BigEndian.Uint32(frameHeader[4:8]))
		if frameSize <= 0 || frameSize > int64(reader.Len()) {
			break
		}

		frameData := make([]byte, frameSize)
		if _, err := reader.Read(frameData); err != nil {
			break
		}

		if (frameID == "USLT" || frameID == "SYLT") && len(frameData) > 5 {
			encoding := frameData[0]
			content := frameData[4:] // skip language 3 bytes
			descEnd := -1
			if encoding == 1 || encoding == 2 {
				for i := 0; i+1 < len(content); i += 2 {
					if content[i] == 0 && content[i+1] == 0 {
						descEnd = i + 2
						break
					}
				}
			} else {
				descEnd = bytes.IndexByte(content, 0)
				if descEnd != -1 {
					descEnd += 1
				}
			}
			var lyricBytes []byte
			if descEnd != -1 && descEnd < len(content) {
				lyricBytes = content[descEnd:]
			} else {
				lyricBytes = content
			}
			toDecode := append([]byte{encoding}, lyricBytes...)
			return decodeID3Text(toDecode)
		}
	}
	return ""
}
