package ui

import (
	"fmt"
	"path/filepath"

	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type CourseSelectionScreen struct {
	manager           *Manager
	courseChecks      map[string]*widget.Check
	courses           []*models.Course
	downloadPath      string
	extensiveCheck    *widget.Check
	overwriteCheck    *widget.Check
	container         *fyne.Container
	courseList        *fyne.Container
	lectureSelections map[string][]int // courseID -> selected lecture indices
}

func getDownloadFolderName() string {
	return fmt.Sprintf("%s-下载", config.PlatformName)
}

func NewCourseSelectionScreen(manager *Manager) fyne.CanvasObject {
	downloadPath := filepath.Join(".", getDownloadFolderName())
	if utils.IsAndroid() {
		// 安卓使用应用存储的temp目录
		downloadPath = "temp"
	}

	cs := &CourseSelectionScreen{
		manager:           manager,
		courseChecks:      make(map[string]*widget.Check),
		lectureSelections: make(map[string][]int),
		downloadPath:      downloadPath,
	}
	cs.loadCourses()
	cs.buildUI()
	return cs.container
}

func (cs *CourseSelectionScreen) loadCourses() {
	progressDialog := dialog.NewProgressInfinite("加载中...", "正在获取课程列表", cs.manager.window)
	progressDialog.Show()

	go func() {
		defer fyne.Do(func() {
			progressDialog.Hide()
		})

		courses, err := cs.manager.apiClient.GetCourseList()
		if err != nil {
			utils.ShowErrorDialog(err, cs.manager.window)
			return
		}
		cs.courses = courses
		fyne.Do(func() {
			cs.updateCourseList()
		})
	}()
}

func (cs *CourseSelectionScreen) buildUI() {
	title := widget.NewLabelWithStyle("选择要下载的课程", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// 返回按钮
	backButton := widget.NewButton("←", func() {
		cs.manager.ShowStudentSelection()
	})
	backButton.Importance = widget.LowImportance

	cs.courseList = container.NewVBox()
	scroll := container.NewScroll(cs.courseList)
	scroll.SetMinSize(fyne.NewSize(600, 400))

	cs.extensiveCheck = widget.NewCheck("下载延伸课程", nil)
	cs.overwriteCheck = widget.NewCheck("覆盖已下载文件", nil)
	scrollContainer := container.NewStack(scroll)

	// 安卓平台不显示路径选择
	var pathContainer fyne.CanvasObject
	if !utils.IsAndroid() {
		pathLabel := widget.NewLabel(fmt.Sprintf("下载路径: %s", cs.downloadPath))
		pathButton := widget.NewButton("选择路径", func() {
			dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil {
					utils.ShowErrorDialog(err, cs.manager.window)
					return
				}
				if uri != nil {
					cs.downloadPath = filepath.Join(uri.Path(), getDownloadFolderName())
					pathLabel.SetText(fmt.Sprintf("下载路径: %s", cs.downloadPath))
				}
			}, cs.manager.window)
		})
		pathContainer = container.NewHBox(
			layout.NewSpacer(),
			pathLabel,
			pathButton,
		)
	} else {
		pathContainer = layout.NewSpacer()
	}

	selectAllButton := widget.NewButton("全选", func() {
		for courseID, check := range cs.courseChecks {
			check.SetChecked(true)
			// 获取对应课程的讲数并全选
			for _, course := range cs.courses {
				if course.CourseID == courseID {
					lectures := make([]int, course.EndLiveNum)
					for i := range lectures {
						lectures[i] = i
					}
					cs.lectureSelections[courseID] = lectures
					break
				}
			}
		}
	})
	deselectAllButton := widget.NewButton("取消全选", func() {
		for courseID, check := range cs.courseChecks {
			check.SetChecked(false)
			cs.lectureSelections[courseID] = []int{}
		}
	})

	downloadButton := widget.NewButton("开始下载", cs.startDownload)
	downloadButton.Importance = widget.HighImportance

	// 顶部部分（标题）
	// 使用Stack布局实现绝对定位，确保标题真正居中
	titleCentered := container.NewHBox(layout.NewSpacer(), title, layout.NewSpacer())
	backButtonContainer := container.NewHBox(backButton, layout.NewSpacer())

	titleRow := container.NewStack(titleCentered, backButtonContainer)
	top := container.NewVBox(
		container.NewPadded(titleRow),
		widget.NewSeparator(),
	)

	// 底部部分（选项与按钮）
	var optionsContainer fyne.CanvasObject
	if !utils.IsAndroid() {
		optionsContainer = container.NewVBox(
			cs.extensiveCheck,
			cs.overwriteCheck,
			pathContainer,
		)
	} else {
		optionsContainer = container.NewVBox(
			cs.extensiveCheck,
		)
	}

	bottom := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(optionsContainer),
		container.NewPadded(
			container.NewHBox(
				selectAllButton,
				deselectAllButton,
				layout.NewSpacer(),
				downloadButton,
			),
		),
	)

	// 中间 + 上下布局
	content := container.NewBorder(
		top, bottom, nil, nil,
		scrollContainer,
	)

	// 包一层 padding，让边缘不贴边
	cs.container = container.NewPadded(content)
}

func (cs *CourseSelectionScreen) updateCourseList() {
	cs.courseList.Objects = nil
	// 添加课程复选框
	for _, course := range cs.courses {
		courseCopy := course // 避免闭包问题

		check := widget.NewCheck(course.SubjectName+" - "+course.CourseName, func(checked bool) {
			if checked && len(cs.lectureSelections[courseCopy.CourseID]) == 0 {
				// 如果勾选但没有选择讲数，默认全选
				lectures := make([]int, courseCopy.EndLiveNum)
				for i := range lectures {
					lectures[i] = i
				}
				cs.lectureSelections[courseCopy.CourseID] = lectures
			} else if !checked {
				// 取消勾选时清空选择
				cs.lectureSelections[courseCopy.CourseID] = []int{}
			}
		})
		cs.courseChecks[course.CourseID] = check

		// 默认全选所有讲
		lectures := make([]int, course.EndLiveNum)
		for i := range lectures {
			lectures[i] = i
		}
		cs.lectureSelections[course.CourseID] = lectures

		// 创建选择讲数的按钮
		selectLecturesBtn := widget.NewButton("...", func() {
			cs.showLectureSelectionDialog(courseCopy)
		})

		// 创建课程行
		courseRow := container.NewBorder(nil, nil, check, selectLecturesBtn)
		cs.courseList.Add(courseRow)
	}
	cs.courseList.Refresh()
}

func (cs *CourseSelectionScreen) showLectureSelectionDialog(course *models.Course) {
	// 创建讲数选择列表
	lectureChecks := make([]*widget.Check, course.EndLiveNum)
	selectedLectures := cs.lectureSelections[course.CourseID]

	// 创建一个map来快速查找已选中的讲
	selectedMap := make(map[int]bool)
	for _, idx := range selectedLectures {
		selectedMap[idx] = true
	}

	for i := 0; i < course.EndLiveNum; i++ {
		lectureChecks[i] = widget.NewCheck(fmt.Sprintf("第%d讲", i+1), nil)
		lectureChecks[i].SetChecked(selectedMap[i])
	}

	// 创建滚动容器
	checkList := container.NewVBox()
	for _, check := range lectureChecks {
		checkList.Add(check)
	}
	scroll := container.NewScroll(checkList)
	scroll.SetMinSize(fyne.NewSize(300, 400))

	// 全选和全不选按钮
	selectAllBtn := widget.NewButton("全选", func() {
		for _, check := range lectureChecks {
			check.SetChecked(true)
		}
	})

	deselectAllBtn := widget.NewButton("全不选", func() {
		for _, check := range lectureChecks {
			check.SetChecked(false)
		}
	})

	// 创建自定义对话框
	var d dialog.Dialog

	confirmBtn := widget.NewButton("确定", func() {
		// 收集选中的讲
		selected := []int{}
		for i, check := range lectureChecks {
			if check.Checked {
				selected = append(selected, i)
			}
		}
		cs.lectureSelections[course.CourseID] = selected

		// 更新主复选框状态
		if check, ok := cs.courseChecks[course.CourseID]; ok {
			if len(selected) == 0 {
				check.SetChecked(false)
			} else if len(selected) == course.EndLiveNum {
				check.SetChecked(true)
			} else {
				// 部分选中状态 - Fyne不支持三态复选框，所以保持勾选但修改文本提示
				check.SetChecked(true)
				check.Text = fmt.Sprintf("%s - %s (已选%d/%d讲)",
					course.SubjectName, course.CourseName, len(selected), course.EndLiveNum)
				check.Refresh()
			}
		}
		d.Hide()
	})

	cancelBtn := widget.NewButton("取消", func() {
		d.Hide()
	})

	buttons := container.NewHBox(
		selectAllBtn,
		deselectAllBtn,
		layout.NewSpacer(),
		cancelBtn,
		confirmBtn,
	)

	content := container.NewBorder(
		widget.NewLabel(fmt.Sprintf("选择要下载的讲 - %s", course.CourseName)),
		buttons,
		nil, nil,
		scroll,
	)

	d = dialog.NewCustomWithoutButtons("选择讲数", content, cs.manager.window)
	d.Resize(fyne.NewSize(400, 500))
	d.Show()
}

func (cs *CourseSelectionScreen) startDownload() {
	var selectedCourses []*models.Course
	selectedLectures := make(map[string][]int)

	for _, course := range cs.courses {
		if check, ok := cs.courseChecks[course.CourseID]; ok && check.Checked {
			if lectures, ok := cs.lectureSelections[course.CourseID]; ok && len(lectures) > 0 {
				selectedCourses = append(selectedCourses, course)
				selectedLectures[course.CourseID] = lectures
			}
		}
	}

	if len(selectedCourses) == 0 {
		dialog.ShowInformation("提示", "未选择任何课程", cs.manager.window)
		return
	}

	if err := utils.Mkdir(cs.downloadPath); err != nil {
		utils.ShowErrorDialog(err, cs.manager.window)
		return
	}

	cs.manager.selectedCourses = selectedCourses
	cs.manager.selectedLectures = selectedLectures
	cs.manager.downloadPath = cs.downloadPath
	cs.manager.isExtensive = cs.extensiveCheck.Checked
	if utils.IsAndroid() {
		cs.manager.isOverwrite = false
	} else {
		cs.manager.isOverwrite = cs.overwriteCheck.Checked
	}
	cs.manager.ShowDownloadProgress()
}
