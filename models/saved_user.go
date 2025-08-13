package models

// SavedUser 保存的用户信息
type SavedUser struct {
	UserID   string `json:"user_id"`  // 用户ID
	Username string `json:"username"` // 用户名（手机号或学员编号）
	Nickname string `json:"nickname"` // 昵称
	Token    string `json:"token"`    // token
	Platform string `json:"platform"` // 平台
}

// SavedUserEncrypted 加密保存的用户信息（用于JSON存储）
type SavedUserEncrypted struct {
	Username          string `json:"username"`           // 用户名（明文）
	EncryptedUserID   string `json:"encrypted_user_id"`  // 加密的用户ID
	EncryptedNickname string `json:"encrypted_nickname"` // 加密的昵称
	EncryptedToken    string `json:"encrypted_token"`    // 加密的token
	Platform          string `json:"platform"`           // 平台（明文）
}

// SavedUsersData 所有保存的用户信息
type SavedUsersData struct {
	Users []SavedUser `json:"users"`
}

// SavedUsersDataEncrypted 所有加密保存的用户信息（用于文件存储）
type SavedUsersDataEncrypted struct {
	Users []SavedUserEncrypted `json:"users"`
}
