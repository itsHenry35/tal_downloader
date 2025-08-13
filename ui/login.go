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
	saveUserCheck  *widget.Check
	loginMode      string
	container      *fyne.Container
	savedUsersData *models.SavedUsersData
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
				d.Dismiss()
			}),
		),
	), ls.manager.window)
	d.Show()
}

// loadSavedUsers 加载保存的用户数据
func (ls *LoginScreen) loadSavedUsers() {
	var err error
	ls.savedUsersData, err = utils.LoadSavedUsers()
	if err != nil {
		ls.savedUsersData = &models.SavedUsersData{Users: []models.SavedUser{}}
	}
}

// showSavedUserSelectionDialog 显示保存用户选择对话框
func (ls *LoginScreen) showSavedUserSelectionDialog() {
	if ls.savedUsersData == nil {
		ls.loadSavedUsers()
	}

	if len(ls.savedUsersData.Users) == 0 {
		utils.ShowInfoDialog("提示", "没有保存的账号", ls.manager.window)
		return
	}

	var selectedUserIndex = -1
	var d dialog.Dialog

	// 创建用户列表项
	var userItems []fyne.CanvasObject
	var checkBoxes []*widget.Check

	for i, user := range ls.savedUsersData.Users {
		// 删除按钮
		deleteBtn := widget.NewButton("-", nil)
		deleteBtn.Importance = widget.DangerImportance

		// 捕获当前用户索引，避免闭包问题
		currentIndex := i
		currentUser := user

		deleteBtn.OnTapped = func() {
			ls.deleteSavedUserFromDialog(currentUser, d)
		}

		// 单选框（显示昵称和平台）
		displayText := fmt.Sprintf("%s - %s", user.Nickname, user.Platform)
		checkBox := widget.NewCheck(displayText, func(checked bool) {
			if checked {
				selectedUserIndex = currentIndex
				// 取消其他复选框的选择
				for j, cb := range checkBoxes {
					if j != currentIndex {
						cb.SetChecked(false)
					}
				}
			} else {
				// 如果取消选择，重置selectedUserIndex
				if selectedUserIndex == currentIndex {
					selectedUserIndex = -1
				}
			}
		})
		checkBoxes = append(checkBoxes, checkBox)

		// 整体布局：左侧单选框 + 右侧删除按钮
		userItem := container.NewBorder(nil, nil, checkBox, deleteBtn, widget.NewLabel(""))
		userItems = append(userItems, userItem)
	}

	// 创建滚动容器
	userList := container.NewVBox(userItems...)
	scrollContainer := container.NewScroll(userList)
	scrollContainer.SetMinSize(fyne.NewSize(500, 300))

	// 确认和取消按钮
	confirmBtn := widget.NewButton("确认", func() {
		if selectedUserIndex >= 0 && selectedUserIndex < len(ls.savedUsersData.Users) {
			d.Dismiss()
			ls.loginWithSavedUser(ls.savedUsersData.Users[selectedUserIndex])
		} else {
			utils.ShowInfoDialog("提示", "请选择一个账号", ls.manager.window)
		}
	})
	confirmBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("取消", func() {
		d.Dismiss()
	})

	buttons := container.NewHBox(
		layout.NewSpacer(),
		cancelBtn,
		confirmBtn,
	)

	content := container.NewBorder(
		widget.NewLabel("选择要使用的账号"),
		buttons,
		nil, nil,
		scrollContainer,
	)

	d = dialog.NewCustomWithoutButtons("选择保存的账号", content, ls.manager.window)
	d.Resize(fyne.NewSize(600, 500))
	d.Show()
}

// loginWithSavedUser 使用保存的用户信息直接登录
func (ls *LoginScreen) loginWithSavedUser(user models.SavedUser) {
	// 设置平台
	config.SetPlatform(user.Platform)

	// 设置认证信息
	ls.manager.apiClient.SetAuth(user.Token, user.UserID)

	// 直接跳转到学员选择页面
	ls.manager.ShowStudentSelection()
}

// deleteSavedUserFromDialog 从对话框中删除保存的用户
func (ls *LoginScreen) deleteSavedUserFromDialog(user models.SavedUser, d dialog.Dialog) {
	dialog.ShowConfirm("确认删除",
		fmt.Sprintf("确定要删除账号 %s (%s/%s) 吗？", user.Nickname, user.Username, user.Platform),
		func(confirmed bool) {
			if confirmed {
				err := utils.RemoveUser(user)
				if err != nil {
					utils.ShowErrorDialog(err, ls.manager.window)
				} else {
					utils.ShowInfoDialog("提示", "账号已删除", ls.manager.window)
					ls.loadSavedUsers() // 重新加载数据
					// 重新显示对话框
					d.Dismiss()
					ls.showSavedUserSelectionDialog()
				}
			}
		},
		ls.manager.window)
}

func (ls *LoginScreen) buildUI() {
	title := widget.NewLabelWithStyle("登录", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	platforms := []string{"乐读", "学而思培优"}
	ls.platformSelect = widget.NewSelect(platforms, func(selected string) {
		config.SetPlatform(selected)
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

	// 保存用户信息复选框
	ls.saveUserCheck = widget.NewCheck("保存用户信息", func(checked bool) {
		ls.manager.isSaveUserInfo = checked
	})
	ls.saveUserCheck.SetChecked(ls.manager.isSaveUserInfo)

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

	// 显示保存用户选择对话框的按钮
	selectSavedUserButton := widget.NewButton("使用保存的账号", func() {
		ls.showSavedUserSelectionDialog()
	})
	selectSavedUserButton.Importance = widget.MediumImportance

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
		selectSavedUserButton,
		loginButton,
	)

	// 主体内容（中间）
	body := container.NewVBox(
		container.NewPadded(platformForm),
		widget.NewSeparator(),
		container.NewPadded(passwordForm),
		container.NewPadded(smsForm),
		container.NewHBox(ls.saveUserCheck),
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
			progressDialog.Dismiss()

			if msgErr != nil {
				utils.ShowErrorDialog(msgErr, ls.manager.window)
				return
			}

			if err != nil {
				utils.ShowErrorDialog(err, ls.manager.window)
				return
			}

			ls.manager.apiClient.SetAuth(authData.Token, authData.UserID)

			// 如果选择保存用户信息，则保存
			if ls.manager.isSaveUserInfo {
				var usernameToSave string
				if loginMode == "password" {
					usernameToSave = username
				} else {
					usernameToSave = phone
				}

				err := utils.AddUser(
					usernameToSave,
					authData.Nickname,
					authData.Token,
					config.PlatformName,
					authData.UserID,
				)
				if err != nil {
					// 保存失败不影响登录流程，只是显示警告
					fmt.Printf("保存用户信息失败: %v\n", err)
				}
			}

			ls.manager.ShowStudentSelection()
		})
	}()
}
