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
