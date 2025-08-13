package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"github.com/itsHenry35/tal_downloader/models"
)

const SavedUsersFileName = "saved_users.json"

// createKey 从username生成32字节的密钥
func createKey(username string) []byte {
	hash := sha256.Sum256([]byte(username))
	return hash[:]
}

// encrypt 使用AES加密数据
func encrypt(data, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 使用AES解密数据
func decrypt(encryptedData string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func getFileURI() fyne.URI {
	return storage.NewFileURI(filepath.Join(GetRootPath(), SavedUsersFileName))
}

// LoadSavedUsers 加载保存的用户信息
func LoadSavedUsers() (*models.SavedUsersData, error) {
	// 创建文件URI
	fileURI := getFileURI()

	// 检查文件是否存在
	exists, err := storage.Exists(fileURI)
	if err != nil {
		return &models.SavedUsersData{Users: []models.SavedUser{}}, err
	}
	if !exists {
		// 文件不存在，创建空文件
		SaveUsers(&models.SavedUsersData{Users: []models.SavedUser{}})
		return &models.SavedUsersData{Users: []models.SavedUser{}}, nil
	}

	// 读取文件
	read, err := storage.Reader(fileURI)
	if err != nil {
		return &models.SavedUsersData{Users: []models.SavedUser{}}, err
	}
	defer read.Close()

	// 尝试解析加密的JSON数据
	var encryptedData models.SavedUsersDataEncrypted
	decoder := json.NewDecoder(read)
	err = decoder.Decode(&encryptedData)
	if err != nil {
		// 如果JSON解析失败，创建空数据
		emptyData := &models.SavedUsersData{Users: []models.SavedUser{}}
		SaveUsers(emptyData)
		return emptyData, nil
	}

	// 解密用户数据
	var users []models.SavedUser
	for _, encUser := range encryptedData.Users {
		key := createKey(encUser.Username)

		// 解密UserID
		userIDBytes, err := decrypt(encUser.EncryptedUserID, key)
		if err != nil {
			continue // 跳过解密失败的用户
		}

		// 解密Nickname
		nicknameBytes, err := decrypt(encUser.EncryptedNickname, key)
		if err != nil {
			continue
		}

		// 解密Token
		tokenBytes, err := decrypt(encUser.EncryptedToken, key)
		if err != nil {
			continue
		}

		user := models.SavedUser{
			Username: encUser.Username,
			UserID:   string(userIDBytes),
			Nickname: string(nicknameBytes),
			Token:    string(tokenBytes),
			Platform: encUser.Platform,
		}
		users = append(users, user)
	}

	return &models.SavedUsersData{Users: users}, nil
}

// SaveUsers 保存用户信息到文件
func SaveUsers(data *models.SavedUsersData) error {
	// 创建文件URI
	fileURI := getFileURI()

	// 加密用户数据
	var encryptedUsers []models.SavedUserEncrypted
	for _, user := range data.Users {
		key := createKey(user.Username)

		// 加密UserID
		encryptedUserID, err := encrypt([]byte(user.UserID), key)
		if err != nil {
			return fmt.Errorf("加密UserID失败: %v", err)
		}

		// 加密Nickname
		encryptedNickname, err := encrypt([]byte(user.Nickname), key)
		if err != nil {
			return fmt.Errorf("加密Nickname失败: %v", err)
		}

		// 加密Token
		encryptedToken, err := encrypt([]byte(user.Token), key)
		if err != nil {
			return fmt.Errorf("加密Token失败: %v", err)
		}

		encUser := models.SavedUserEncrypted{
			Username:          user.Username,
			EncryptedUserID:   encryptedUserID,
			EncryptedNickname: encryptedNickname,
			EncryptedToken:    encryptedToken,
			Platform:          user.Platform,
		}
		encryptedUsers = append(encryptedUsers, encUser)
	}

	encryptedData := &models.SavedUsersDataEncrypted{Users: encryptedUsers}

	// 创建写入器
	write, err := storage.Writer(fileURI)
	if err != nil {
		return fmt.Errorf("创建文件写入器失败: %v", err)
	}
	defer write.Close()

	// 编码为JSON并写入
	encoder := json.NewEncoder(write)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(encryptedData)
	if err != nil {
		return fmt.Errorf("保存用户数据失败: %v", err)
	}

	return nil
}

// AddUser 添加用户到保存列表
func AddUser(username, nickname, token, platform, userID string) error {
	data, err := LoadSavedUsers()
	if err != nil {
		return err
	}

	// 检查用户是否已存在（通过用户名和平台判断）
	for i, user := range data.Users {
		if user.Username == username && user.Platform == platform {
			// 更新现有用户信息
			data.Users[i].Nickname = nickname
			data.Users[i].Token = token
			data.Users[i].UserID = userID
			return SaveUsers(data)
		}
	}

	// 添加新用户
	newUser := models.SavedUser{
		Username: username,
		Nickname: nickname,
		Token:    token,
		Platform: platform,
		UserID:   userID,
	}

	data.Users = append(data.Users, newUser)
	return SaveUsers(data)
}

// RemoveUser 从保存列表中移除用户
func RemoveUser(user models.SavedUser) error {
	data, err := LoadSavedUsers()
	if err != nil {
		return err
	}

	// 查找并移除用户
	for i, u := range data.Users {
		if u.Username == user.Username && u.Platform == user.Platform {
			// 移除这个用户
			data.Users = append(data.Users[:i], data.Users[i+1:]...)
			return SaveUsers(data)
		}
	}

	return fmt.Errorf("未找到要删除的用户")
}

// GetUser 根据用户名和平台获取保存的用户信息
func GetUser(username, platform string) (*models.SavedUser, error) {
	data, err := LoadSavedUsers()
	if err != nil {
		return nil, err
	}

	for _, user := range data.Users {
		if user.Username == username && user.Platform == platform {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("未找到用户")
}
