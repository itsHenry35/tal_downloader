package api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/models"
)

// LoginWithPassword performs password login
func (c *Client) LoginWithPassword(username, password string) (*models.AuthData, error) {
	// First try 100tal login
	loginURL := fmt.Sprintf("%s/v1/web/login/pwd", config.PassportAPIBase)

	formData := url.Values{}
	formData.Set("symbol", username)
	formData.Set("password", password)
	formData.Set("source_type", "2")
	formData.Set("domain", "xueersi.com")

	headers := map[string]string{
		"content-type": "application/x-www-form-urlencoded",
		"ver-num":      "1.13.03",
		"client-id":    config.ClientID,
		"device-id":    config.DeviceID,
	}

	resp, err := c.doRequest("POST", loginURL, formData, headers, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.AccountLoginResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode == 0 {
		return c.getFinalAuth(result.Data.Code)
	} else {
		err = fmt.Errorf("%s", result.ErrMsg)
	}

	// Try direct course API login
	studentIdLoginResult, errn := c.loginWithStudentId(username, password)
	if errn != nil {
		// we tend to return the first error
		return nil, err
	}
	return studentIdLoginResult, nil
}

// SendSMSCode sends SMS verification code
func (c *Client) SendSMSCode(phone, zoneCode string) error {
	sendURL := fmt.Sprintf("%s/v1/web/login/sms/send", config.PassportAPIBase)

	formData := url.Values{}
	formData.Set("verify_type", "1")
	formData.Set("phone", phone)
	formData.Set("phone_code", zoneCode)

	headers := map[string]string{
		"content-type": "application/x-www-form-urlencoded; charset=UTF-8",
		"ver-num":      "1.13.03",
		"client-id":    config.ClientID,
		"device-id":    config.DeviceID,
		"origin":       "owcr://classroom",
	}

	resp, err := c.doRequest("POST", sendURL, formData, headers, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("%s", result.ErrMsg)
	}

	return nil
}

// LoginWithSMS performs SMS login
func (c *Client) LoginWithSMS(phone, smsCode, zoneCode string) (*models.AuthData, error) {
	loginURL := fmt.Sprintf("%s/v1/web/login/sms", config.PassportAPIBase)

	formData := url.Values{}
	formData.Set("phone", phone)
	formData.Set("sms_code", smsCode)
	formData.Set("phone_code", zoneCode)
	formData.Set("source_type", "2")
	formData.Set("domain", "xueersi.com")

	headers := map[string]string{
		"content-type": "application/x-www-form-urlencoded",
		"ver-num":      "1.13.03",
		"client-id":    config.ClientID,
		"device-id":    config.DeviceID,
	}

	resp, err := c.doRequest("POST", loginURL, formData, headers, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.AccountLoginResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("%s", result.ErrMsg)
	}

	return c.getFinalAuth(result.Data.Code)
}

func (c *Client) loginWithStudentId(username, password string) (*models.AuthData, error) {
	loginURL := fmt.Sprintf("%s/passport/v1/login/student/password", config.CourseAPIBase)

	body := map[string]string{
		"account":  username,
		"password": password,
		"deviceId": config.DeviceID,
		"clientId": config.ClientID,
	}

	resp, err := c.doRequest("POST", loginURL, body, nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var authResponse models.AuthFinalResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, err
	}

	authData := models.AuthData{
		Token:  authResponse.Token,
		UserID: fmt.Sprint(authResponse.UserID),
	}

	return &authData, nil
}

func (c *Client) getFinalAuth(code string) (*models.AuthData, error) {
	finalAuthURL := fmt.Sprintf("%s/passport/v1/login/student/code", config.CourseAPIBase)

	body := map[string]string{
		"code":     code,
		"deviceId": config.DeviceID,
		"terminal": config.Terminal,
		"product":  "ss",
		"clientId": config.ClientID,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := c.doRequest("POST", finalAuthURL, body, headers, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var authResponse models.AuthFinalResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, err
	}

	authData := models.AuthData{
		Token:  authResponse.Token,
		UserID: fmt.Sprint(authResponse.UserID),
	}

	return &authData, nil
}

// GetStudentAccounts 获取当前账号下的所有学生账号列表
func (c *Client) GetStudentAccounts() (models.StudentAccountListResponse, error) {
	url := fmt.Sprintf("%s/passport/v1/students/account-list", config.CourseAPIBase)

	payload := map[string]string{
		"stuPuId":   c.userID, // 从客户端已登录信息中取
		"signToken": c.token,  // 同上
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Origin":       "owcr://classroom",
		"Referer":      "https://speiyou.cn/",
		"resVer":       "1.0.6",
		"terminal":     "pc",
		"token":        c.token,
		"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		"version":      "3.22.0.99",
	}

	resp, err := c.doRequest("POST", url, payload, headers, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.StudentAccountListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// SwitchStudentAccount 切换学生账号
func (c *Client) SwitchStudentAccount(currentUID, nextUID string) error {
	url := fmt.Sprintf("%s/passport/v2/login/student/change-stu", config.CourseAPIBase)

	payload := map[string]string{
		"stuPuId":        nextUID,
		"currentStuPuId": currentUID,
		"signToken":      c.token,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"terminal":     "pc",
		"token":        c.token,
		"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		"version":      "3.22.0.99",
	}

	resp, err := c.doRequest("POST", url, payload, headers, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result models.AuthFinalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.SetAuth(result.Token, fmt.Sprint(result.UserID))
	return nil
}
