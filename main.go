package main

import (
	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/ui"
	"github.com/itsHenry35/tal_downloader/utils"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

func main() {
	myApp := app.New()

	// 安卓平台启动时清理临时文件夹
	if utils.IsAndroid() {
		utils.CleanAndroidTempFolder()
	}

	window := myApp.NewWindow("好未来课程下载器")
	window.Resize(config.DefaultWindowSize)

	// Create main container
	mainContainer := container.NewStack()

	// Initialize UI manager
	uiManager := ui.NewManager(window, mainContainer)

	// Start with login screen
	uiManager.ShowLogin()

	window.SetContent(mainContainer)
	window.ShowAndRun()
}
