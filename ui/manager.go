package ui

import (
	"os"

	"github.com/itsHenry35/tal_downloader/api"
	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/downloader"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

type Manager struct {
	window               fyne.Window
	mainContainer        *fyne.Container
	apiClient            *api.Client
	downloader           *downloader.Downloader
	authData             *models.AuthData
	selectedCourses      []*models.Course
	selectedLectures     map[string][]int // courseID -> selected lecture indices
	downloadPath         string
	isExtensive          bool
	isOverwrite          bool
	currentScreen        string
	isConfirmScreenShown bool
}

func NewManager(window fyne.Window, mainContainer *fyne.Container) *Manager {
	manager := &Manager{
		window:               window,
		mainContainer:        mainContainer,
		apiClient:            api.NewClient(),
		downloader:           downloader.NewDownloader(config.MaxConcurrentDownloads, config.ThreadCount),
		selectedLectures:     make(map[string][]int),
		currentScreen:        "login",
		isConfirmScreenShown: false,
	}

	// 设置安卓返回键处理
	if utils.IsAndroid() {
		window.Canvas().SetOnTypedKey(manager.handleAndroidBackKey)
	}

	return manager
}

// handleAndroidBackKey 处理安卓返回键事件
func (m *Manager) handleAndroidBackKey(keyEvent *fyne.KeyEvent) {
	if keyEvent.Name != mobile.KeyBack || m.isConfirmScreenShown {
		return
	}

	switch m.currentScreen {
	case "login":
		// 登录页面，确认退出
		m.isConfirmScreenShown = true
		utils.ShowCustomConfirm("退出应用", "确定", "取消",
			container.NewVBox(widget.NewLabel("确定要退出应用吗？")),
			func(confirmed bool) {
				m.isConfirmScreenShown = false
				if confirmed {
					os.Exit(0)
				}
			}, m.window)
	case "student":
		// 学员选择页面，确认返回登录
		m.isConfirmScreenShown = true
		utils.ShowCustomConfirm("返回登录", "确定", "取消",
			container.NewVBox(widget.NewLabel("确定要返回登录页面吗？")),
			func(confirmed bool) {
				m.isConfirmScreenShown = false
				if confirmed {
					m.ShowLogin()
				}
			}, m.window)
	case "course":
		// 课程选择页面，确认返回学员选择
		m.isConfirmScreenShown = true
		utils.ShowCustomConfirm("返回上级", "确定", "取消",
			container.NewVBox(widget.NewLabel("确定要返回学员选择页面吗？")),
			func(confirmed bool) {
				m.isConfirmScreenShown = false
				if confirmed {
					m.ShowStudentSelection()
				}
			}, m.window)
	case "download":
		// 下载进度页面，确认返回课程选择
		m.isConfirmScreenShown = true
		utils.ShowCustomConfirm("返回上级", "确定", "取消",
			container.NewVBox(widget.NewLabel("确定要返回课程选择页面吗？")),
			func(confirmed bool) {
				m.isConfirmScreenShown = false
				if confirmed {
					m.ShowCourseSelection()
				}
			}, m.window)
	}
}

func (m *Manager) ShowLogin() {
	m.currentScreen = "login"
	loginScreen := NewLoginScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{loginScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowStudentSelection() {
	m.currentScreen = "student"
	studentScreen := NewStudentSelectScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{studentScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowCourseSelection() {
	m.currentScreen = "course"
	courseScreen := NewCourseSelectionScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{courseScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowDownloadProgress() {
	m.currentScreen = "download"
	downloadScreen := NewDownloadProgressScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{downloadScreen}
	m.mainContainer.Refresh()
}
