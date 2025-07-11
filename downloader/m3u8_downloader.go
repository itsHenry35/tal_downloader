package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/itsHenry35/tal_downloader/utils"
)

func (d *Downloader) downloadM3U8(task *DownloadTask) error {
	task.Status = "preparing"
	task.StartTime = time.Now()

	// 创建临时目录
	tmpDir := filepath.Join(filepath.Dir(task.FilePath), fmt.Sprintf(".tmp_%d_%s", time.Now().UnixNano(), filepath.Base(task.FilePath)))

	// 使用安卓安全的目录创建
	if err := utils.Mkdir(tmpDir); err != nil {
		return err
	}
	defer func() {
		// 清理临时目录
		if utils.IsAndroid() {
			actualPath := utils.GetAndroidSafeFilePath(tmpDir)
			os.RemoveAll(actualPath)
		} else {
			os.RemoveAll(tmpDir)
		}
	}()

	// 获取m3u8内容
	resp, err := d.client.Get(task.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	lines := strings.Split(string(body), "\n")
	var tsList []string
	baseURL := task.URL[:strings.LastIndex(task.URL, "/")+1]
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if !strings.HasPrefix(line, "http") {
				line = baseURL + line
			}
			tsList = append(tsList, line)
		}
	}

	task.TotalSize = int64(len(tsList))
	task.Status = "downloading"

	var wg sync.WaitGroup
	concurrency := d.perFileThreads
	sem := make(chan struct{}, concurrency)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastTime := time.Now()
	lastDownloaded := int64(0)

	go func() {
		for range ticker.C {
			if task.Status != "downloading" {
				return
			}
			now := time.Now()
			completed := atomic.LoadInt64(&task.DownloadedParts)
			percent := float64(completed) / float64(task.TotalSize) * 100
			speed := float64(completed-lastDownloaded) / now.Sub(lastTime).Seconds()
			task.progress(percent*0.9, fmt.Sprintf("%.2f ts/s", speed), atomic.LoadInt64(&task.Downloaded), -1)
			lastTime = now
			lastDownloaded = completed
		}
	}()

	for idx, tsURL := range tsList {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			filePath := filepath.Join(tmpDir, fmt.Sprintf("%05d.ts", i))
			for attempt := 0; attempt < 3; attempt++ {
				err := downloadTS(d.client, url, filePath, task)
				if err == nil {
					atomic.AddInt64(&task.DownloadedParts, 1)
					return
				}
				time.Sleep(time.Second)
			}
		}(idx, tsURL)
	}

	wg.Wait()

	task.Status = "merging"
	if task.progress != nil {
		task.progress(90, "合并中", task.Downloaded, -1)
	}

	return mergeTSFiles(tmpDir, task.FilePath, task)
}

func downloadTS(client *http.Client, url, filePath string, task *DownloadTask) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := utils.CreateFile(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err == nil {
		atomic.AddInt64(&task.Downloaded, n)
	}
	return err
}

func mergeTSFiles(tmpDir, outputFile string, task *DownloadTask) error {
	out, err := utils.CreateFile(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// 获取实际的临时目录路径
	actualTmpDir := tmpDir
	if utils.IsAndroid() {
		actualTmpDir = utils.GetAndroidSafeFilePath(tmpDir)
	}

	files, err := os.ReadDir(actualTmpDir)
	if err != nil {
		return err
	}

	// 按名字排序，确保顺序
	var tsFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".ts") {
			tsFiles = append(tsFiles, f.Name())
		}
	}
	sort.Strings(tsFiles)

	for _, name := range tsFiles {
		path := filepath.Join(actualTmpDir, name)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		if err != nil {
			return err
		}
	}
	if err := out.Sync(); err != nil {
		return err
	}

	// 获取最终文件大小
	actualOutputPath := outputFile
	if utils.IsAndroid() {
		actualOutputPath = utils.GetAndroidSafeFilePath(outputFile)
	}

	stat, err := os.Stat(actualOutputPath)
	if err != nil {
		task.progress(100, "Completed", -1, atomic.LoadInt64(&task.Downloaded))
		return nil
	}
	finalFileSize := stat.Size()
	task.progress(100, "Completed", -1, finalFileSize)
	return nil
}
