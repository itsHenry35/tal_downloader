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
	Error           error

	progress   func(float64, string, int64, int64)
	cancelFunc func()
	isPaused   atomic.Bool
	wg         sync.WaitGroup

	// 进度更新相关
	progressMutex    sync.Mutex
	lastProgressTime time.Time
	lastDownloaded   int64

	// 状态相关
	statusMutex sync.RWMutex
	status      string
}

// Status getter and setter methods for thread safety
func (task *DownloadTask) Status() string {
	task.statusMutex.RLock()
	defer task.statusMutex.RUnlock()
	return task.status
}

func (task *DownloadTask) SetStatus(status string) {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	task.status = status
}

type ProgressManager struct {
	tasks      map[*DownloadTask]bool
	tasksMutex sync.RWMutex
	ticker     *time.Ticker
	stopChan   chan struct{}
	started    bool
	startMutex sync.Mutex // 保护started字段
}

func NewProgressManager() *ProgressManager {
	return &ProgressManager{
		tasks:    make(map[*DownloadTask]bool),
		stopChan: make(chan struct{}),
	}
}

func (pm *ProgressManager) AddTask(task *DownloadTask) {
	pm.tasksMutex.Lock()
	defer pm.tasksMutex.Unlock()

	task.progressMutex.Lock()
	task.lastProgressTime = time.Now()
	task.lastDownloaded = 0
	task.progressMutex.Unlock()

	pm.tasks[task] = true

	pm.startMutex.Lock()
	shouldStart := !pm.started
	if shouldStart {
		pm.started = true
	}
	pm.startMutex.Unlock()

	if shouldStart {
		pm.start()
	}
}

func (pm *ProgressManager) RemoveTask(task *DownloadTask) {
	pm.tasksMutex.Lock()
	defer pm.tasksMutex.Unlock()

	delete(pm.tasks, task)

	pm.startMutex.Lock()
	shouldStop := len(pm.tasks) == 0 && pm.started
	if shouldStop {
		pm.started = false
	}
	pm.startMutex.Unlock()

	if shouldStop {
		pm.stop()
	}
}

func (pm *ProgressManager) start() {
	pm.ticker = time.NewTicker(100 * time.Millisecond)

	go func() {
		defer pm.ticker.Stop()
		for {
			select {
			case <-pm.ticker.C:
				pm.updateAllTasks()
			case <-pm.stopChan:
				return
			}
		}
	}()
}

func (pm *ProgressManager) stop() {
	// 创建新的stopChan来避免重复关闭问题
	pm.startMutex.Lock()
	oldStopChan := pm.stopChan
	pm.stopChan = make(chan struct{})
	pm.startMutex.Unlock()

	// 通知停止（非阻塞）
	select {
	case oldStopChan <- struct{}{}:
	default:
	}
}

func (pm *ProgressManager) updateAllTasks() {
	pm.tasksMutex.RLock()
	taskList := make([]*DownloadTask, 0, len(pm.tasks))
	for task := range pm.tasks {
		taskList = append(taskList, task)
	}
	pm.tasksMutex.RUnlock()

	now := time.Now()
	for _, task := range taskList {
		if task.progress == nil {
			continue
		}

		status := task.Status()
		if status != "downloading" && status != "merging" {
			continue
		}

		pm.updateSingleTask(task, now, status)
	}
}

func (pm *ProgressManager) updateSingleTask(task *DownloadTask, now time.Time, status string) {
	task.progressMutex.Lock()
	defer task.progressMutex.Unlock()

	if task.TotalSize < 0 {
		// M3U8下载的进度计算（TotalSize为负数存储总段数）
		totalParts := -task.TotalSize

		if status == "merging" {
			// 合并阶段显示90%进度
			if task.progress != nil {
				task.progress(90, "合并中", atomic.LoadInt64(&task.Downloaded), -1)
			}
			return
		}

		// 下载阶段的进度计算
		completed := atomic.LoadInt64(&task.DownloadedParts)

		if task.lastProgressTime.IsZero() {
			task.lastProgressTime = now
			task.lastDownloaded = completed
			return
		}

		timeDiff := now.Sub(task.lastProgressTime).Seconds()
		if timeDiff > 0 {
			speed := float64(completed-task.lastDownloaded) / timeDiff
			percent := float64(completed) / float64(totalParts) * 90 // 最多到90%，留10%给合并
			if task.progress != nil {
				task.progress(percent, fmt.Sprintf("%.2f ts/s (%d/%d)", speed, completed, totalParts), atomic.LoadInt64(&task.Downloaded), -1)
			}
			task.lastProgressTime = now
			task.lastDownloaded = completed
		}
	} else if task.TotalSize > 0 {
		// 普通文件下载的进度计算
		downloaded := atomic.LoadInt64(&task.Downloaded)
		if task.lastProgressTime.IsZero() {
			task.lastProgressTime = now
			task.lastDownloaded = downloaded
			return
		}

		timeDiff := now.Sub(task.lastProgressTime).Seconds()
		if timeDiff > 0 {
			bytes := downloaded - task.lastDownloaded
			speed := float64(bytes) / timeDiff / 1024 / 1024
			progress := float64(downloaded) / float64(task.TotalSize) * 100
			if task.progress != nil {
				task.progress(progress, fmt.Sprintf("%.2f MB/s", speed), downloaded, task.TotalSize)
			}
			task.lastProgressTime = now
			task.lastDownloaded = downloaded
		}
	}
}

type Downloader struct {
	concurrentFiles int
	perFileThreads  int
	tasks           []*DownloadTask
	mu              sync.Mutex
	client          *http.Client
	progressManager *ProgressManager
}

func NewDownloader(concurrentFiles, perFileThreads int) *Downloader {
	transport := &http.Transport{
		MaxIdleConns:        512,
		MaxIdleConnsPerHost: 512,
		MaxConnsPerHost:     512,
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
		progressManager: NewProgressManager(),
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
		progress: progressFunc,
	}
	task.SetStatus("pending")
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
				t.SetStatus("error")
				if t.progress != nil {
					t.progress(0, fmt.Sprintf("错误： %v", err), -1, -1)
				}
			}
		}(task)
	}
}

func (d *Downloader) downloadRegularFile(task *DownloadTask) error {
	task.SetStatus("preparing")
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

	task.SetStatus("downloading")

	// 将任务添加到进度管理器
	d.progressManager.AddTask(task)
	defer d.progressManager.RemoveTask(task)

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
		task.SetStatus("completed")
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

	task.SetStatus("downloading")
	task.StartTime = time.Now()

	// 将任务添加到进度管理器
	d.progressManager.AddTask(task)
	defer d.progressManager.RemoveTask(task)

	buf := make([]byte, 32*1024)

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

	task.SetStatus("completed")
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
	task.SetStatus("cancelled")
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
