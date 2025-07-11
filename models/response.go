package models

type AccountLoginResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    struct {
		Code string `json:"code"`
	} `json:"data"`
}

type AuthFinalResponse struct {
	Token  string `json:"hb_token"`
	UserID int    `json:"pu_uid"`
}

type StudentAccountListResponse []*StudentAccount

type VideoUrlResponse struct {
	VideoURLs []string `json:"videoUrls"`
	Message   string   `json:"message"`
}

type RecordModeVideoUrlResponse struct {
	Definitions map[string][]string `json:"definitions"`
	Message     string              `json:"message"`
}
