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
	task.SetStatus("preparing")
	task.StartTime = time.Now()

	// 创建临时目录
	tmpDir := filepath.Join(filepath.Dir(task.FilePath), fmt.Sprintf(".tmp_%d_%s", time.Now().UnixNano(), filepath.Base(task.FilePath)))

	// 使用安卓安全的目录创建
	if err := utils.Mkdir(tmpDir); err != nil {
		return err
	}

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
	task.SetStatus("downloading")

	// 使用负数存储总段数，便于进度管理器识别M3U8任务
	task.TotalSize = -int64(len(tsList))

	// 将任务添加到进度管理器
	d.progressManager.AddTask(task)

	var wg sync.WaitGroup
	concurrency := d.perFileThreads
	sem := make(chan struct{}, concurrency)

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

	// 进入合并阶段，进度管理器会自动显示90%进度
	task.SetStatus("merging")

	// 合并TS文件
	err = mergeTSFiles(tmpDir, task.FilePath)

	// 清理临时目录（这也是合并过程的一部分）
	if utils.IsAndroid() {
		actualPath := utils.GetAndroidSafeFilePath(tmpDir)
		os.RemoveAll(actualPath)
	} else {
		os.RemoveAll(tmpDir)
	}

	// 合并和清理都完成后，先从进度管理器移除任务，再设置完成状态
	d.progressManager.RemoveTask(task)

	if err == nil {
		task.SetStatus("completed")
		// 手动发送最终完成进度
		if task.progress != nil {
			// 获取最终文件大小
			actualOutputPath := task.FilePath
			if utils.IsAndroid() {
				actualOutputPath = utils.GetAndroidSafeFilePath(task.FilePath)
			}

			stat, statErr := os.Stat(actualOutputPath)
			if statErr == nil {
				task.progress(100, "Completed", -1, stat.Size())
			} else {
				task.progress(100, "Completed", -1, atomic.LoadInt64(&task.Downloaded))
			}
		}
	}

	return err
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

	// 使用缓冲读取以支持暂停功能
	buf := make([]byte, 32*1024)
	var totalBytes int64

	for {
		// 检查暂停状态
		if task.isPaused.Load() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	atomic.AddInt64(&task.Downloaded, totalBytes)
	return nil
}

func mergeTSFiles(tmpDir, outputFile string) error {
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
	return out.Sync()
}
