package ui

import (
	"github.com/itsHenry35/tal_downloader/api"
	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/downloader"
	"github.com/itsHenry35/tal_downloader/models"

	"fyne.io/fyne/v2"
)

type Manager struct {
	window           fyne.Window
	mainContainer    *fyne.Container
	apiClient        *api.Client
	downloader       *downloader.Downloader
	authData         *models.AuthData
	selectedCourses  []*models.Course
	selectedLectures map[string][]int // courseID -> selected lecture indices
	downloadPath     string
	isExtensive      bool
	isOverwrite      bool
}

func NewManager(window fyne.Window, mainContainer *fyne.Container) *Manager {
	return &Manager{
		window:           window,
		mainContainer:    mainContainer,
		apiClient:        api.NewClient(),
		downloader:       downloader.NewDownloader(config.MaxConcurrentDownloads, config.ThreadCount),
		selectedLectures: make(map[string][]int),
	}
}

func (m *Manager) ShowLogin() {
	loginScreen := NewLoginScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{loginScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowStudentSelection() {
	studentScreen := NewStudentSelectScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{studentScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowCourseSelection() {
	courseScreen := NewCourseSelectionScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{courseScreen}
	m.mainContainer.Refresh()
}

func (m *Manager) ShowDownloadProgress() {
	downloadScreen := NewDownloadProgressScreen(m)
	m.mainContainer.Objects = []fyne.CanvasObject{downloadScreen}
	m.mainContainer.Refresh()
}
