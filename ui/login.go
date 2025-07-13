package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/constants"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"
)

type LoginScreen struct {
	manager        *Manager
	platformSelect *widget.Select
	usernameEntry  *widget.Entry
	passwordEntry  *widget.Entry
	phoneEntry     *widget.Entry
	smsCodeEntry   *widget.Entry
	zoneSelect     *widget.Select
	sendButton     *widget.Button
	loginMode      string
	container      *fyne.Container
}

func NewLoginScreen(manager *Manager) fyne.CanvasObject {
	ls := &LoginScreen{
		manager:   manager,
		loginMode: "password",
	}
	ls.buildUI()
	return ls.container
}

func (ls *LoginScreen) showVersionDialog() {
	info := fmt.Sprintf(
		"版本号: %s\n编译时间: %s\n作者: %s",
		constants.Version,
		constants.BuildTime,
		constants.Author,
	)

	var d dialog.Dialog

	d = dialog.NewCustomWithoutButtons("关于", container.NewVBox(
		widget.NewLabel(info),
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewButton("GitHub", func() {
				utils.OpenURL(constants.GithubURL)

			}),
			widget.NewButton("反馈问题", func() {
				dialog.ShowCustomConfirm("反馈问题",
					"继续", "取消",
					container.NewVBox(
						widget.NewLabel("请确认您当前使用的程序版本为最新版本。"),
						widget.NewLabel("若非最新版本，请先升级至最新版本后，再进行问题测试。"),
						widget.NewLabel("有关版本发布详情，请参阅 GitHub 页面。"),
						widget.NewLabel("如问题仍未解决，请点击“继续”并在浏览器中提交 Issue。"),
					),
					func(confirmed bool) {
						if confirmed {
							utils.OpenURL(constants.FeedbackURL)
						}
					},
					ls.manager.window,
				)
			}),
			widget.NewButton("关闭", func() {
				d.Hide()
			}),
		),
	), ls.manager.window)
	d.Show()
}

func (ls *LoginScreen) buildUI() {
	title := widget.NewLabelWithStyle("登录", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	platforms := []string{"乐读", "学而思培优"}
	platformMap := map[string]string{
		"乐读":    "ledu",
		"学而思培优": "xes",
	}
	ls.platformSelect = widget.NewSelect(platforms, func(selected string) {
		config.SetPlatform(platformMap[selected])
	})
	ls.platformSelect.SetSelected("乐读") // 默认选择乐读

	platformForm := container.NewVBox(
		widget.NewLabel("请选择平台:"),
		ls.platformSelect,
	)

	ls.usernameEntry = widget.NewEntry()
	ls.usernameEntry.SetPlaceHolder("手机号或学员编号")
	ls.passwordEntry = widget.NewPasswordEntry()
	ls.passwordEntry.SetPlaceHolder("密码")

	passwordForm := container.NewVBox(
		widget.NewLabel("用户名:"),
		ls.usernameEntry,
		widget.NewLabel("密码:"),
		ls.passwordEntry,
	)

	ls.phoneEntry = widget.NewEntry()
	ls.phoneEntry.SetPlaceHolder("手机号")
	ls.smsCodeEntry = widget.NewEntry()
	ls.smsCodeEntry.SetPlaceHolder("验证码")

	zones := []string{"中国 +86", "中国台湾 +886", "中国澳门 +853", "中国香港 +852"}
	ls.zoneSelect = widget.NewSelect(zones, nil)
	ls.zoneSelect.SetSelected("中国 +86")
	ls.sendButton = widget.NewButton("发送验证码", ls.sendSMSCode)

	smsForm := container.NewVBox(
		widget.NewLabel("区号:"),
		ls.zoneSelect,
		widget.NewLabel("手机号:"),
		ls.phoneEntry,
		widget.NewLabel("验证码:"),
		container.NewBorder(nil, nil, nil, ls.sendButton, ls.smsCodeEntry),
	)
	smsForm.Hide()

	var switchToSMS, switchToPwd *widget.Button

	switchToSMS = widget.NewButton("短信验证码登录", func() {
		ls.loginMode = "sms"
		passwordForm.Hide()
		switchToSMS.Hide()
		switchToPwd.Show()
		smsForm.Show()
		ls.phoneEntry.SetText(ls.usernameEntry.Text)
	})

	switchToPwd = widget.NewButton("账号密码登录", func() {
		ls.loginMode = "password"
		smsForm.Hide()
		switchToPwd.Hide()
		switchToSMS.Show()
		passwordForm.Show()
		ls.usernameEntry.SetText(ls.phoneEntry.Text)
	})
	switchToPwd.Hide() // 初始隐藏短信登录按钮

	versionButton := widget.NewButton(constants.Version, func() {
		ls.showVersionDialog()
	})
	versionButton.Importance = widget.LowImportance
	if strings.Contains(constants.Version, "PR") || strings.Contains(constants.Version, "CI") || strings.Contains(constants.Version, "Debug") {
		versionButton.Importance = widget.DangerImportance
	}

	loginButton := widget.NewButton("登录", ls.doLogin)
	loginButton.Importance = widget.HighImportance

	// 按钮区域（底部，右对齐）
	footer := container.NewHBox(
		versionButton,
		layout.NewSpacer(),
		loginButton,
	)

	// 主体内容（中间）
	body := container.NewVBox(
		container.NewPadded(platformForm),
		widget.NewSeparator(),
		container.NewPadded(passwordForm),
		container.NewPadded(smsForm),
		container.NewHBox(layout.NewSpacer(), switchToSMS, switchToPwd, layout.NewSpacer()),
	)

	// 页面整体布局：顶部标题 + 中间内容 + 底部按钮
	content := container.NewBorder(
		container.NewVBox(
			container.NewPadded(title),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewPadded(footer),
		),
		nil, nil,
		body,
	)

	// 统一padding
	ls.container = container.NewPadded(content)
}

func (ls *LoginScreen) sendSMSCode() {
	phone := ls.phoneEntry.Text
	if phone == "" {
		utils.ShowErrorDialog(fmt.Errorf("请输入手机号"), ls.manager.window)
		return
	}

	zoneMap := map[string]string{
		"中国 +86":    "86",
		"中国台湾 +886": "886",
		"中国澳门 +853": "853",
		"中国香港 +852": "852",
	}
	zoneCode := zoneMap[ls.zoneSelect.Selected]

	go func() {
		err := ls.manager.apiClient.SendSMSCode(phone, zoneCode)
		if err != nil {
			utils.ShowErrorDialog(err, ls.manager.window)
			return
		}
		utils.ShowInfoDialog("提示", "验证码已发送", ls.manager.window)
		fyne.Do(func() {
			ls.sendButton.Disable()
		})
		for i := 60; i > 0; i-- {
			fyne.Do(func() {
				ls.sendButton.SetText(fmt.Sprintf("重新发送(%d秒)", i))
			})
			time.Sleep(time.Second)
		}
		fyne.Do(func() {
			ls.sendButton.SetText("发送验证码")
			ls.sendButton.Enable()
		})
	}()
}

func (ls *LoginScreen) doLogin() {
	progressDialog := dialog.NewProgressInfinite("登录中...", "正在验证身份信息", ls.manager.window)
	progressDialog.Show()

	go func() {
		var (
			authData *models.AuthData
			err      error
			msgErr   error
		)

		loginMode := ls.loginMode
		username := ls.usernameEntry.Text
		password := ls.passwordEntry.Text
		phone := ls.phoneEntry.Text
		smsCode := ls.smsCodeEntry.Text
		zone := ls.zoneSelect.Selected

		if loginMode == "password" {
			if username == "" || password == "" {
				msgErr = fmt.Errorf("请填写用户名和密码")
			} else {
				authData, err = ls.manager.apiClient.LoginWithPassword(username, password)
			}
		} else {
			if phone == "" || smsCode == "" {
				msgErr = fmt.Errorf("请填写手机号和验证码")
			} else {
				zoneMap := map[string]string{
					"中国 +86":    "86",
					"中国台湾 +886": "886",
					"中国澳门 +853": "853",
					"中国香港 +852": "852",
				}
				zoneCode := zoneMap[zone]
				authData, err = ls.manager.apiClient.LoginWithSMS(phone, smsCode, zoneCode)
			}
		}

		fyne.Do(func() {
			progressDialog.Hide()

			if msgErr != nil {
				utils.ShowErrorDialog(msgErr, ls.manager.window)
				return
			}

			if err != nil {
				utils.ShowErrorDialog(err, ls.manager.window)
				return
			}

			ls.manager.authData = authData
			ls.manager.apiClient.SetAuth(authData.Token, authData.UserID)
			ls.manager.ShowStudentSelection()
		})
	}()
}
