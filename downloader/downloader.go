package downloader

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/itsHenry35/tal_downloader/utils"
)

type DownloadTask struct {
	URL             string
	FilePath        string
	TotalSize       int64
	Downloaded      int64
	DownloadedParts int64 //used in m3u8 downloads
	StartTime       time.Time
	Status          string
	Error           error

	progress   func(float64, string, int64, int64)
	cancelFunc func()
	isPaused   atomic.Bool
	wg         sync.WaitGroup
}

type Downloader struct {
	concurrentFiles int
	perFileThreads  int
	tasks           []*DownloadTask
	mu              sync.Mutex
	client          *http.Client
}

func NewDownloader(concurrentFiles, perFileThreads int) *Downloader {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Downloader{
		concurrentFiles: concurrentFiles,
		perFileThreads:  perFileThreads,
		client: &http.Client{
			Timeout:   0,
			Transport: transport,
		},
	}
}

func (d *Downloader) AddTask(url, filePath string, progressFunc func(float64, string, int64, int64)) *DownloadTask {
	task := &DownloadTask{
		URL:      url,
		FilePath: filePath,
		Status:   "pending",
		progress: progressFunc,
	}
	d.mu.Lock()
	d.tasks = append(d.tasks, task)
	d.mu.Unlock()
	return task
}

func (d *Downloader) Start() {
	semaphore := make(chan struct{}, d.concurrentFiles)

	for _, task := range d.tasks {
		task.wg.Add(1)
		go func(t *DownloadTask) {
			defer t.wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			if err := d.downloadFile(t); err != nil {
				t.Error = err
				t.Status = "error"
				if t.progress != nil {
					t.progress(0, fmt.Sprintf("错误： %v", err), -1, -1)
				}
			}
		}(task)
	}
}

func (d *Downloader) downloadRegularFile(task *DownloadTask) error {
	task.Status = "preparing"
	task.StartTime = time.Now()

	// 创建目录
	if err := utils.Mkdir(filepath.Dir(task.FilePath)); err != nil {
		return err
	}

	// HEAD 请求判断是否支持 Range
	req, err := http.NewRequest("HEAD", task.URL, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	supportsRange := strings.ToLower(resp.Header.Get("Accept-Ranges")) == "bytes"
	if !supportsRange {
		// 回退到单线程下载
		task.TotalSize = -1 // 标记为未知大小
		return d.downloadSingleThread(task)
	}
	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		return fmt.Errorf("Content-Length not provided")
	}
	task.TotalSize, _ = strconv.ParseInt(sizeStr, 10, 64)

	return d.downloadMultiThread(task)
}

func (d *Downloader) downloadMultiThread(task *DownloadTask) error {
	file, err := utils.CreateFile(task.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 预分配文件大小
	if err := file.Truncate(task.TotalSize); err != nil {
		return err
	}

	partSize := task.TotalSize / int64(d.perFileThreads)
	var wg sync.WaitGroup

	task.Status = "downloading"

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastTime := time.Now()
	lastDownloaded := int64(0)

	go func() {
		for range ticker.C {
			if task.Status != "downloading" {
				return
			}
			if task.progress != nil && task.TotalSize > 0 {
				now := time.Now()
				bytes := atomic.LoadInt64(&task.Downloaded) - lastDownloaded
				sec := now.Sub(lastTime).Seconds()
				speed := float64(bytes) / sec / 1024 / 1024
				progress := float64(task.Downloaded) / float64(task.TotalSize) * 100
				task.progress(progress, fmt.Sprintf("%.2f MB/s", speed), atomic.LoadInt64(&task.Downloaded), atomic.LoadInt64(&task.TotalSize))
				lastTime = now
				lastDownloaded = atomic.LoadInt64(&task.Downloaded)
			}
		}
	}()

	for i := 0; i < d.perFileThreads; i++ {
		wg.Add(1)
		start := int64(i) * partSize
		end := start + partSize - 1
		if i == d.perFileThreads-1 {
			end = task.TotalSize - 1
		}

		go func(start, end int64) {
			defer wg.Done()
			req, err := http.NewRequest("GET", task.URL, nil)
			if err != nil {
				task.Error = err
				return
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

			resp, err := d.client.Do(req)
			if err != nil {
				task.Error = err
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 32*1024)
			offset := start
			for {
				if task.isPaused.Load() {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, err = file.WriteAt(buf[:n], offset)
					if err != nil {
						task.Error = err
						return
					}
					offset += int64(n)
					atomic.AddInt64(&task.Downloaded, int64(n))
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					task.Error = err
					return
				}
			}
		}(start, end)
	}

	wg.Wait()

	if task.Error == nil {
		task.Status = "completed"
		if task.progress != nil {
			task.progress(100, "Completed", atomic.LoadInt64(&task.TotalSize), atomic.LoadInt64(&task.TotalSize))
		}
	}
	return task.Error
}

func (d *Downloader) downloadSingleThread(task *DownloadTask) error {
	resp, err := d.client.Get(task.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := utils.CreateFile(task.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	task.Status = "downloading"
	task.StartTime = time.Now()

	buf := make([]byte, 32*1024)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastTime := time.Now()
	lastDownloaded := int64(0)

	go func() {
		for range ticker.C {
			if task.Status != "downloading" {
				return
			}
			if task.progress != nil && task.TotalSize > 0 {
				now := time.Now()
				bytes := atomic.LoadInt64(&task.Downloaded) - lastDownloaded
				sec := now.Sub(lastTime).Seconds()
				speed := float64(bytes) / sec / 1024 / 1024
				progress := float64(task.Downloaded) / float64(task.TotalSize) * 100
				task.progress(progress, fmt.Sprintf("%.2f MB/s", speed), atomic.LoadInt64(&task.Downloaded), atomic.LoadInt64(&task.TotalSize))
				lastTime = now
				lastDownloaded = atomic.LoadInt64(&task.Downloaded)
			}
		}
	}()

	for {
		if task.isPaused.Load() {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, err := file.Write(buf[:n])
			if err != nil {
				return err
			}
			atomic.AddInt64(&task.Downloaded, int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	task.Status = "completed"
	if task.progress != nil {
		task.progress(100, "Completed", task.Downloaded, task.TotalSize)
	}
	return nil
}

func (task *DownloadTask) Pause() {
	task.isPaused.Store(true)
}

func (task *DownloadTask) Resume() {
	task.isPaused.Store(false)
}

func (task *DownloadTask) Cancel() {
	if task.cancelFunc != nil {
		task.cancelFunc()
	}
	task.Status = "cancelled"
}

func (task *DownloadTask) Wait() {
	task.wg.Wait()
}

func (d *Downloader) downloadFile(task *DownloadTask) error {
	if strings.Contains(strings.ToLower(task.URL), ".m3u8") {
		return d.downloadM3U8(task)
	}
	// fallback 原本的下载器
	return d.downloadRegularFile(task)
}
