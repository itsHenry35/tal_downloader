package utils

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func ShowErrorDialog(err error, window fyne.Window) {
	fyne.Do(func() {
		dialog.ShowError(err, window)
	})
}

func ShowInfoDialog(title, message string, window fyne.Window) {
	fyne.Do(func() {
		dialog.ShowInformation(title, message, window)
	})
}

func ShowCustomConfirm(title string, confirm string, dismiss string, content fyne.CanvasObject, callback func(bool), parent fyne.Window) {
	fyne.Do(func() {
		dialog.ShowCustomConfirm(title, confirm, dismiss, content, func(ok bool) {
			callback(ok)
		}, parent)
	})
}
