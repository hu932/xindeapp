package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var dumpMu sync.Mutex

// saveUploadDump：上传详情落盘（异步，不影响响应速度）
func saveUploadDump(id string, detail map[string]any) {
	if id == "" {
		id = dateTimeStamp() + "_" + md5short(jsonEncode(detail))
	}
	go func() {
		dumpMu.Lock()
		defer dumpMu.Unlock()
		dir := dp("upload_dumps")
		_ = os.MkdirAll(dir, 0o755)
		detail["time"] = firstNonNilStr(detail["time"], tsDateTime())
		_ = os.WriteFile(filepath.Join(dir, id+".json"), []byte(jsonEncode(detail)), 0o644)
		// 概率清理，保留最近 50 个
		if timeNowUnix()%50 == 0 {
			pruneDir(dir, 50)
		}
	}()
}

func pruneDir(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e)
		}
	}
	if len(files) <= keep {
		return
	}
	// 按修改时间排序，删除最旧
	sortDirEntriesByModTime(files, dir)
	for i := 0; i < len(files)-keep; i++ {
		_ = os.Remove(filepath.Join(dir, files[i].Name()))
	}
}

func sortDirEntriesByModTime(files []os.DirEntry, dir string) {
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			ti, _ := os.Stat(filepath.Join(dir, files[i].Name()))
			tj, _ := os.Stat(filepath.Join(dir, files[j].Name()))
			if ti != nil && tj != nil && ti.ModTime().After(tj.ModTime()) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}
