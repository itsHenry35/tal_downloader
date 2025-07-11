package ui

import (
	"fmt"

	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type StudentSelectScreen struct {
	manager      *Manager
	studentList  models.StudentAccountListResponse
	selected     *models.StudentAccount
	container    *fyne.Container
	radioGroup   *widget.RadioGroup
	switchButton *widget.Button
}

func NewStudentSelectScreen(manager *Manager) fyne.CanvasObject {
	sl := &StudentSelectScreen{
		manager: manager,
	}
	sl.loadStudents()
	sl.buildUI()
	return sl.container
}

func (sl *StudentSelectScreen) loadStudents() {
	progressDialog := dialog.NewProgressInfinite("加载中...", "正在获取学员列表", sl.manager.window)
	progressDialog.Show()

	go func() {
		accounts, err := sl.manager.apiClient.GetStudentAccounts()

		// 所有 UI 操作集中到 fyne.Do，避免 UI 线程冲突
		fyne.Do(func() {
			progressDialog.Hide()

			if err != nil {
				utils.ShowErrorDialog(err, sl.manager.window)
				return
			}

			sl.studentList = accounts

			// 默认选中当前账号
			_, currentUID := sl.manager.apiClient.GetAuth()
			defaultIndex := 0
			options := make([]string, len(accounts))
			for i, acc := range accounts {
				options[i] = acc.Nickname
				if fmt.Sprint(acc.PuUID) == currentUID {
					defaultIndex = i
				}
			}

			sl.radioGroup.Options = options
			sl.radioGroup.Selected = options[defaultIndex]
			sl.selected = accounts[defaultIndex]
			sl.radioGroup.Refresh()
		})
	}()
}

func (sl *StudentSelectScreen) buildUI() {
	title := widget.NewLabelWithStyle("请选择学员", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	sl.radioGroup = widget.NewRadioGroup([]string{}, func(selected string) {
		for i, option := range sl.radioGroup.Options {
			if selected == option {
				sl.selected = sl.studentList[i]
				break
			}
		}
	})

	// 滚动区
	scroll := container.NewScroll(sl.radioGroup)
	scroll.SetMinSize(fyne.NewSize(400, 300)) // 初始高度
	scrollContainer := container.NewStack(scroll)

	// 底部按钮
	sl.switchButton = widget.NewButton("选择账号", sl.switchAccount)
	sl.switchButton.Importance = widget.HighImportance
	bottom := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
			layout.NewSpacer(),
			sl.switchButton,
		),
	)

	// 顶部标题
	top := container.NewVBox(
		container.NewPadded(title),
		widget.NewSeparator(),
	)

	// 主内容结构
	content := container.NewBorder(
		top,
		bottom,
		nil, nil,
		scrollContainer,
	)

	// 外层加 padding
	sl.container = container.NewPadded(content)
}

func (sl *StudentSelectScreen) switchAccount() {
	// 在主线程中读取状态，防止 goroutine 访问 UI
	var selectedUID string
	var currentUID string
	var needSwitch bool

	_, currentUID = sl.manager.apiClient.GetAuth()
	if sl.selected != nil {
		selectedUID = fmt.Sprint(sl.selected.PuUID)
		needSwitch = selectedUID != currentUID
	}

	if !needSwitch {
		sl.manager.ShowCourseSelection()
		return
	}

	progressDialog := dialog.NewProgressInfinite("切换中...", "正在切换账号", sl.manager.window)
	progressDialog.Show()

	go func() {
		err := sl.manager.apiClient.SwitchStudentAccount(currentUID, selectedUID)

		// 所有 UI 更新统一在主线程处理
		fyne.Do(func() {
			progressDialog.Hide()

			if err != nil {
				utils.ShowErrorDialog(err, sl.manager.window)
				return
			}

			sl.manager.apiClient.SetAuth("", selectedUID)
			sl.manager.ShowCourseSelection()
		})
	}()
}
