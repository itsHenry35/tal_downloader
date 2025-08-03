package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itsHenry35/tal_downloader/downloader"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type ProgressUpdate struct {
	filePath  string
	progress  float64
	speed     string
	currSize  int64
	totalSize int64
}

type DownloadProgressScreen struct {
	manager            *Manager
	progressBars       map[string]*widget.ProgressBar
	speedLabels        map[string]*widget.Label
	sizeLabels         map[string]*widget.Label
	saveButtons        map[string]*widget.Button           // 新增：保存按钮映射
	pauseResumeButtons map[string]*widget.Button           // 新增：每个任务的暂停/继续按钮映射
	taskMap            map[string]*downloader.DownloadTask // 新增：文件路径到任务的映射
	downloadTasks      []*downloader.DownloadTask
	pauseButton        *widget.Button
	isPaused           bool
	container          *fyne.Container
	progressList       *fyne.Container
	courseContainers   map[string]*fyne.Container
	courseFoldState    map[string]bool
	courseFoldButtons  map[string]*widget.Button

	// 批量更新相关
	updateChannel  chan ProgressUpdate
	pendingUpdates map[string]ProgressUpdate
	updateMutex    sync.Mutex

	// 线程安全相关
	tasksMutex  sync.RWMutex
	uiMapsMutex sync.RWMutex
}

func (ds *DownloadProgressScreen) toggleCourseFold(courseID string) {
	if box, ok := ds.courseContainers[courseID]; ok {
		ds.courseFoldState[courseID] = !ds.courseFoldState[courseID]
		if ds.courseFoldState[courseID] {
			box.Show()
		} else {
			box.Hide()
		}
		if btn, ok := ds.courseFoldButtons[courseID]; ok {
			if ds.courseFoldState[courseID] {
				btn.SetText("-")
			} else {
				btn.SetText("+")
			}
		}
	}
}

func NewDownloadProgressScreen(manager *Manager) fyne.CanvasObject {
	ds := &DownloadProgressScreen{
		manager:            manager,
		progressBars:       make(map[string]*widget.ProgressBar),
		speedLabels:        make(map[string]*widget.Label),
		sizeLabels:         make(map[string]*widget.Label),
		saveButtons:        make(map[string]*widget.Button),
		pauseResumeButtons: make(map[string]*widget.Button),
		taskMap:            make(map[string]*downloader.DownloadTask),
		courseContainers:   make(map[string]*fyne.Container),
		courseFoldState:    make(map[string]bool),
		courseFoldButtons:  make(map[string]*widget.Button),
		updateChannel:      make(chan ProgressUpdate, 1000), // 缓冲通道
		pendingUpdates:     make(map[string]ProgressUpdate),
	}

	// 启动批量更新协程
	go ds.batchUpdateHandler()

	ds.buildUI()
	ds.startDownloads()
	return ds.container
}

// batchUpdateHandler 批量处理进度更新
func (ds *DownloadProgressScreen) batchUpdateHandler() {
	ticker := time.NewTicker(100 * time.Millisecond) // 每100毫秒批量更新一次
	defer ticker.Stop()

	for {
		select {
		case update := <-ds.updateChannel:
			// 收集更新到pending map
			ds.updateMutex.Lock()
			ds.pendingUpdates[update.filePath] = update
			ds.updateMutex.Unlock()

		case <-ticker.C:
			// 定时处理所有pending更新
			ds.updateMutex.Lock()
			if len(ds.pendingUpdates) > 0 {
				updates := make(map[string]ProgressUpdate)
				for k, v := range ds.pendingUpdates {
					updates[k] = v
				}
				// 清空pending
				ds.pendingUpdates = make(map[string]ProgressUpdate)
				ds.updateMutex.Unlock()

				// 批量应用UI更新
				fyne.Do(func() {
					for _, update := range updates {
						ds.applyProgressUpdate(update)
					}
				})
			} else {
				ds.updateMutex.Unlock()
			}
		}
	}
}

// applyProgressUpdate 实际应用进度更新到UI
func (ds *DownloadProgressScreen) applyProgressUpdate(update ProgressUpdate) {
	filePath := update.filePath
	progress := update.progress
	speed := update.speed
	currSize := update.currSize
	totalSize := update.totalSize

	// 使用读锁保护对UI maps的访问
	ds.uiMapsMutex.RLock()
	bar, hasBar := ds.progressBars[filePath]
	speedLabel, hasSpeedLabel := ds.speedLabels[filePath]
	sizeLabel, hasSizeLabel := ds.sizeLabels[filePath]
	pauseBtn, hasPauseBtn := ds.pauseResumeButtons[filePath]
	saveBtn, hasSaveBtn := ds.saveButtons[filePath]
	ds.uiMapsMutex.RUnlock()

	if hasBar {
		bar.SetValue(progress / 100)
	}

	if hasSpeedLabel {
		if progress >= 100 {
			speedLabel.SetText("已完成")
			// 隐藏暂停/继续按钮（下载成功）
			if hasPauseBtn {
				pauseBtn.Hide()
			}
			// 安卓平台显示保存按钮
			if utils.IsAndroid() && hasSaveBtn {
				saveBtn.Show()
			}
		} else {
			if strings.Contains(speed, "错误") {
				speedLabel.SetText(speed)
				speedLabel.Importance = widget.DangerImportance
				// 下载失败时，将暂停按钮改为继续
				if hasPauseBtn {
					pauseBtn.SetText("继续")
				}
			} else {
				speedLabel.SetText(fmt.Sprintf("下载速度: %s", speed))
			}
		}
	}

	if hasSizeLabel {
		if progress >= 100 {
			sizeLabel.SetText(fmt.Sprintf("大小: %s", utils.FormatFileSize(totalSize)))
		} else {
			sizeLabel.SetText(fmt.Sprintf("已下载: %s / %s", utils.FormatFileSize(currSize), utils.FormatFileSize(totalSize)))
		}
	}
}

func (ds *DownloadProgressScreen) buildUI() {
	title := widget.NewLabelWithStyle("下载进度", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// 滚动区
	ds.progressList = container.NewVBox()
	scroll := container.NewScroll(ds.progressList)
	scroll.SetMinSize(fyne.NewSize(800, 400))     // 给个初始高度防止太小
	scrollContainer := container.NewStack(scroll) // ✅ 统一使用 container.NewMax

	// 底部按钮
	ds.pauseButton = widget.NewButton("暂停全部", ds.togglePause)

	var footer *fyne.Container
	if !utils.IsAndroid() {
		openFolderButton := widget.NewButton("打开下载目录", func() {
			err := utils.OpenDirectory(ds.manager.downloadPath)
			if err != nil {
				dialog.ShowError(fmt.Errorf("无法打开目录: %v", err), ds.manager.window)
			}
		})
		footer = container.NewHBox(
			ds.pauseButton,
			layout.NewSpacer(),
			openFolderButton,
		)
	} else {
		footer = container.NewHBox(
			ds.pauseButton,
			layout.NewSpacer(),
		)
	}

	// 上部标题
	top := container.NewVBox(
		container.NewPadded(title),
		widget.NewSeparator(),
	)

	// 下部按钮区域
	bottom := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(footer),
	)

	// 主布局：上下 + 中间滚动
	content := container.NewBorder(
		top, bottom, nil, nil,
		scrollContainer,
	)

	// 外层加统一 Padding
	ds.container = container.NewPadded(content)
}

func (ds *DownloadProgressScreen) startDownloads() {
	progressList := ds.progressList
	dl := ds.manager.downloader

	var wg sync.WaitGroup

	for i, course := range ds.manager.selectedCourses {
		courseName := fmt.Sprintf("%s - %s", course.SubjectName, course.CourseName)
		safeName := utils.SanitizeFileName(courseName)

		var courseDir string
		if utils.IsAndroid() {
			// 安卓使用相对路径
			courseDir = filepath.Join("temp", safeName)
		} else {
			courseDir = filepath.Join(ds.manager.downloadPath, safeName)
		}

		if i != 0 {
			progressList.Add(widget.NewSeparator())
		}

		// 获取选中的讲
		selectedLectureIndices := ds.manager.selectedLectures[course.CourseID]
		selectedCount := len(selectedLectureIndices)

		// 添加课程标题
		courseLabel := widget.NewLabelWithStyle(
			fmt.Sprintf("课程 %d/%d: %s (下载%d讲)", i+1, len(ds.manager.selectedCourses), safeName, selectedCount),
			fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		btn := widget.NewButton("-", func() {
			ds.toggleCourseFold(course.CourseID)
		})
		ds.courseFoldButtons[course.CourseID] = btn
		header := container.NewHBox(
			btn,
			courseLabel,
		)
		courseBox := container.NewVBox()
		ds.courseContainers[course.CourseID] = courseBox
		ds.courseFoldState[course.CourseID] = true // 默认展开

		progressList.Add(container.NewVBox(
			header,
			courseBox,
			widget.NewSeparator(),
		))

		wg.Add(1)
		go func(course *models.Course, courseDir string, selectedIndices []int) {
			defer wg.Done()

			lectures, err := ds.manager.apiClient.GetLectures(course.CourseID, course.TutorID)
			if err != nil {
				utils.ShowErrorDialog(err, ds.manager.window)
				return
			}

			// 创建选中索引的map以便快速查找
			selectedMap := make(map[int]bool)
			for _, idx := range selectedIndices {
				selectedMap[idx] = true
			}

			// 只下载选中的讲，并且不超过endLiveNum
			for j, lecture := range lectures {
				// 跳过未选中的讲
				if !selectedMap[j] {
					continue
				}

				// 确保不超过已结束的讲数
				if j >= course.EndLiveNum {
					fyne.Do(func() {
						ds.addErrorItem(course.CourseID, fmt.Sprintf("第%d讲", j+1), "该讲尚未开始")
					})
					continue
				}

				fileName := fmt.Sprintf("第%d讲.mp4", j+1)
				if ds.manager.isExtensive {
					lecture.LiveTypeString = "ONLINE_REAL_RECORD" // 强制设为延伸课程类型
					fileName = fmt.Sprintf("第%d讲_延伸内容.mp4", j+1)
				}
				filePath := filepath.Join(courseDir, fileName)

				// 检查文件是否存在
				if !utils.IsAndroid() {
					if utils.IsFileExists(filePath) && !ds.manager.isOverwrite {
						fyne.Do(func() {
							ds.addProgressItem(course.CourseID, fileName, filePath, true, -1)
						})
						continue
					}
				}

				videoURL, err := ds.manager.apiClient.GetVideoURL(lecture, course.CourseID, course.TutorID)
				if err != nil {
					fyne.Do(func() {
						ds.addErrorItem(course.CourseID, fileName, err.Error())
					})
					continue
				}

				task := dl.AddTask(videoURL, filePath, func(progress float64, speed string, currsize int64, totalSize int64) {
					ds.updateProgress(filePath, progress, speed, currsize, totalSize)
				})

				// 线程安全地添加任务
				ds.tasksMutex.Lock()
				ds.downloadTasks = append(ds.downloadTasks, task)
				ds.taskMap[filePath] = task // 保存任务映射
				ds.tasksMutex.Unlock()

				fyne.Do(func() {
					ds.addProgressItem(course.CourseID, fileName, filePath, false, task.TotalSize)
				})
			}
		}(course, courseDir, selectedLectureIndices)

	}

	// 等待所有任务添加完成后，只启动一次下载器
	go func() {
		wg.Wait()
		dl.Start()
	}()

	progressList.Refresh()
}

func (ds *DownloadProgressScreen) addProgressItem(courseID, fileName, filePath string, exists bool, totalSize int64) {
	progress := widget.NewProgressBar()

	fileLabel := widget.NewLabel(fileName)
	speedLabel := widget.NewLabel("等待中...")
	sizeLabel := widget.NewLabel(fmt.Sprintf("大小: %s", utils.FormatFileSize(totalSize)))
	if exists {
		progress.SetValue(1.0)
		speedLabel.SetText("文件已存在")
	}

	// 创建单独的暂停/继续按钮
	pauseResumeButton := widget.NewButton("暂停", func() {
		ds.toggleSingleTask(filePath)
	})
	if exists {
		pauseResumeButton.Hide() // 文件已存在时隐藏按钮
	}

	// 为安卓创建保存按钮
	var saveButton *widget.Button
	if utils.IsAndroid() {
		saveButton = widget.NewButton("保存", func() {
			ds.saveFileToAndroid(filePath, fileName)
		})
		saveButton.Hide() // 初始隐藏，下载完成后显示
		saveButton.Importance = widget.HighImportance
	}

	// 线程安全地添加到maps
	ds.uiMapsMutex.Lock()
	ds.progressBars[filePath] = progress
	ds.speedLabels[filePath] = speedLabel
	ds.sizeLabels[filePath] = sizeLabel
	ds.pauseResumeButtons[filePath] = pauseResumeButton
	if saveButton != nil {
		ds.saveButtons[filePath] = saveButton
	}
	ds.uiMapsMutex.Unlock()

	// 创建速度标签容器
	var speedContainer fyne.CanvasObject
	if utils.IsAndroid() && saveButton != nil {
		speedContainer = container.NewHBox(speedLabel, layout.NewSpacer(), pauseResumeButton, saveButton)
	} else {
		speedContainer = container.NewHBox(speedLabel, layout.NewSpacer(), pauseResumeButton)
	}

	item := container.NewVBox(
		container.NewHBox(fileLabel, layout.NewSpacer(), sizeLabel),
		progress,
		speedContainer,
	)

	if courseBox, ok := ds.courseContainers[courseID]; ok {
		courseBox.Add(item)
		courseBox.Refresh()
		return
	}
}

func (ds *DownloadProgressScreen) addErrorItem(courseID, fileName, errorMsg string) {
	fileLabel := widget.NewLabel(fileName)
	errorLabel := widget.NewLabel(errorMsg)
	errorLabel.Importance = widget.DangerImportance

	item := container.NewVBox(
		container.NewHBox(fileLabel, layout.NewSpacer(), errorLabel),
	)

	if courseBox, ok := ds.courseContainers[courseID]; ok {
		courseBox.Add(item)
		courseBox.Refresh()
		return
	}
}

func (ds *DownloadProgressScreen) updateProgress(filePath string, progress float64, speed string, currSize int64, totalSize int64) {
	// 发送更新到通道，避免阻塞
	select {
	case ds.updateChannel <- ProgressUpdate{
		filePath:  filePath,
		progress:  progress,
		speed:     speed,
		currSize:  currSize,
		totalSize: totalSize,
	}:
	default:
		// 如果通道满了，跳过这次更新
	}
}

func (ds *DownloadProgressScreen) togglePause() {
	ds.isPaused = !ds.isPaused

	// 线程安全地访问任务列表
	ds.tasksMutex.RLock()
	tasks := make([]*downloader.DownloadTask, len(ds.downloadTasks))
	copy(tasks, ds.downloadTasks)
	taskMapCopy := make(map[string]*downloader.DownloadTask)
	for k, v := range ds.taskMap {
		taskMapCopy[k] = v
	}
	ds.tasksMutex.RUnlock()

	// 线程安全地访问按钮map
	ds.uiMapsMutex.RLock()
	buttonsCopy := make(map[string]*widget.Button)
	for k, v := range ds.pauseResumeButtons {
		buttonsCopy[k] = v
	}
	ds.uiMapsMutex.RUnlock()

	if ds.isPaused {
		ds.pauseButton.SetText("继续全部")
		for _, task := range tasks {
			task.Pause()
		}
		// 更新所有单独按钮的状态
		for filePath, btn := range buttonsCopy {
			if task, ok := taskMapCopy[filePath]; ok && (task.Status() == "downloading" || task.Status() == "preparing") {
				btn.SetText("继续")
			}
		}
	} else {
		ds.pauseButton.SetText("暂停全部")
		for _, task := range tasks {
			task.Resume()
		}
		// 更新所有单独按钮的状态
		for filePath, btn := range buttonsCopy {
			if task, ok := taskMapCopy[filePath]; ok && (task.Status() == "downloading" || task.Status() == "preparing") {
				btn.SetText("暂停")
			}
		}
	}
}

// toggleSingleTask 处理单个任务的暂停/继续
func (ds *DownloadProgressScreen) toggleSingleTask(filePath string) {
	ds.tasksMutex.RLock()
	task, ok := ds.taskMap[filePath]
	ds.tasksMutex.RUnlock()

	if ok {
		ds.uiMapsMutex.RLock()
		btn, hasBbtn := ds.pauseResumeButtons[filePath]
		ds.uiMapsMutex.RUnlock()

		if hasBbtn {
			// 检查任务当前是否被暂停
			if btn.Text == "暂停" {
				task.Pause()
				btn.SetText("继续")
			} else {
				task.Resume()
				btn.SetText("暂停")
			}
		}
	}
}

// saveFileToAndroid 处理安卓平台的文件保存
func (ds *DownloadProgressScreen) saveFileToAndroid(tempPath, fileName string) {
	// 显示保存对话框
	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, ds.manager.window)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		// 使用工具函数复制文件
		if err := utils.CopyToAndroidStorage(tempPath, writer); err != nil {
			dialog.ShowError(err, ds.manager.window)
			return
		}

		// 保存成功后删除临时文件
		actualPath := utils.GetAndroidSafeFilePath(tempPath)
		if err := os.Remove(actualPath); err != nil {
			// 删除失败不影响主流程，只记录日志
			fmt.Printf("警告：删除临时文件失败: %v\n", err)
		}

		// 更新按钮状态
		ds.uiMapsMutex.RLock()
		saveBtn, hasSaveBtn := ds.saveButtons[tempPath]
		ds.uiMapsMutex.RUnlock()

		if hasSaveBtn {
			saveBtn.SetText("已保存")
			saveBtn.Disable()
		}

		dialog.ShowInformation("成功", "文件已保存到存储", ds.manager.window)
	}, ds.manager.window)

	// 设置默认文件名
	saveDialog.SetFileName(fileName)
	saveDialog.Show()
}
