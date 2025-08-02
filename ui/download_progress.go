package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsHenry35/tal_downloader/downloader"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type DownloadProgressScreen struct {
	manager           *Manager
	progressBars      map[string]*widget.ProgressBar
	speedLabels       map[string]*widget.Label
	sizeLabels        map[string]*widget.Label
	saveButtons       map[string]*widget.Button // 新增：保存按钮映射
	downloadTasks     []*downloader.DownloadTask
	pauseButton       *widget.Button
	isPaused          bool
	container         *fyne.Container
	progressList      *fyne.Container
	courseContainers  map[string]*fyne.Container
	courseFoldState   map[string]bool
	courseFoldButtons map[string]*widget.Button
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
		manager:           manager,
		progressBars:      make(map[string]*widget.ProgressBar),
		speedLabels:       make(map[string]*widget.Label),
		sizeLabels:        make(map[string]*widget.Label),
		saveButtons:       make(map[string]*widget.Button),
		courseContainers:  make(map[string]*fyne.Container),
		courseFoldState:   make(map[string]bool),
		courseFoldButtons: make(map[string]*widget.Button),
	}
	ds.buildUI()
	ds.startDownloads()
	return ds.container
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

		go func(course *models.Course, courseDir string, selectedIndices []int) {
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
					fyne.Do(func() {
						ds.updateProgress(filePath, progress, speed, currsize, totalSize)
					})
				})

				ds.downloadTasks = append(ds.downloadTasks, task)
				fyne.Do(func() {
					ds.addProgressItem(course.CourseID, fileName, filePath, false, task.TotalSize)
				})
			}
			dl.Start()
		}(course, courseDir, selectedLectureIndices)

	}
	progressList.Refresh()
}

func (ds *DownloadProgressScreen) addProgressItem(courseID, fileName, filePath string, exists bool, totalSize int64) {
	progress := widget.NewProgressBar()
	ds.progressBars[filePath] = progress

	fileLabel := widget.NewLabel(fileName)
	speedLabel := widget.NewLabel("等待中...")
	sizeLabel := widget.NewLabel(fmt.Sprintf("大小: %s", utils.FormatFileSize(totalSize)))
	if exists {
		progress.SetValue(1.0)
		speedLabel.SetText("文件已存在")
	}
	ds.speedLabels[filePath] = speedLabel
	ds.sizeLabels[filePath] = sizeLabel

	// 为安卓创建保存按钮
	var saveButton *widget.Button
	if utils.IsAndroid() {
		saveButton = widget.NewButton("保存", func() {
			ds.saveFileToAndroid(filePath, fileName)
		})
		saveButton.Hide() // 初始隐藏，下载完成后显示
		saveButton.Importance = widget.HighImportance
		ds.saveButtons[filePath] = saveButton
	}

	// 创建速度标签容器
	var speedContainer fyne.CanvasObject
	if utils.IsAndroid() && saveButton != nil {
		speedContainer = container.NewHBox(speedLabel, layout.NewSpacer(), saveButton)
	} else {
		speedContainer = speedLabel
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
	if bar, ok := ds.progressBars[filePath]; ok {
		bar.SetValue(progress / 100)
	}
	if label, ok := ds.speedLabels[filePath]; ok {
		if progress >= 100 {
			label.SetText("已完成")
			// 安卓平台显示保存按钮
			if utils.IsAndroid() {
				if saveBtn, ok := ds.saveButtons[filePath]; ok {
					saveBtn.Show()
				}
			}
		} else {
			if strings.Contains(speed, "错误") {
				label.SetText(speed)
				label.Importance = widget.DangerImportance
			} else {
				label.SetText(fmt.Sprintf("下载速度: %s", speed))
			}
		}
	}
	if sizeLabel, ok := ds.sizeLabels[filePath]; ok {
		if progress >= 100 {
			sizeLabel.SetText(fmt.Sprintf("大小: %s", utils.FormatFileSize(totalSize)))
		} else {
			sizeLabel.SetText(fmt.Sprintf("已下载: %s / %s", utils.FormatFileSize(currSize), utils.FormatFileSize(totalSize)))
		}
	}
}

func (ds *DownloadProgressScreen) togglePause() {
	ds.isPaused = !ds.isPaused
	if ds.isPaused {
		ds.pauseButton.SetText("继续全部")
		for _, task := range ds.downloadTasks {
			task.Pause()
		}
	} else {
		ds.pauseButton.SetText("暂停全部")
		for _, task := range ds.downloadTasks {
			task.Resume()
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
		if saveBtn, ok := ds.saveButtons[tempPath]; ok {
			saveBtn.SetText("已保存")
			saveBtn.Disable()
		}

		dialog.ShowInformation("成功", "文件已保存到存储", ds.manager.window)
	}, ds.manager.window)

	// 设置默认文件名
	saveDialog.SetFileName(fileName)
	saveDialog.Show()
}
