package utils

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
)

func IsAndroid() bool {
	return runtime.GOOS == "android"
}

func CleanAndroidTempFolder() {
	_ = os.RemoveAll(GetAndroidSafeFilePath("temp"))
}

func GetAndroidSafeFilePath(relativePath string) string {
	if !IsAndroid() {
		return relativePath
	}

	app := fyne.CurrentApp()
	// 获取应用的内部存储路径
	rootPath := app.Storage().RootURI().Path()
	return filepath.Join(rootPath, relativePath)
}

func Mkdir(path string) error {
	if !IsAndroid() {
		return os.MkdirAll(path, 0755)
	}

	// 对于安卓，获取实际路径
	actualPath := GetAndroidSafeFilePath(path)
	return os.MkdirAll(actualPath, 0755)
}

func IsFileExists(path string) bool {
	if !IsAndroid() {
		_, err := os.Stat(path)
		return err == nil
	}

	actualPath := GetAndroidSafeFilePath(path)
	_, err := os.Stat(actualPath)
	return err == nil
}

func CreateFile(path string) (*os.File, error) {
	if !IsAndroid() {
		return os.Create(path)
	}

	actualPath := GetAndroidSafeFilePath(path)
	// 确保目录存在
	dir := filepath.Dir(actualPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}

	return os.Create(actualPath)
}

func CopyToAndroidStorage(sourcePath string, writer fyne.URIWriteCloser) error {
	actualPath := GetAndroidSafeFilePath(sourcePath)

	// 打开源文件
	source, err := os.Open(actualPath)
	if err != nil {
		return fmt.Errorf("无法打开源文件: %v", err)
	}
	defer source.Close()

	// 复制文件
	_, err = io.Copy(writer, source)
	if err != nil {
		return fmt.Errorf("复制文件失败: %v", err)
	}

	return nil
}

func OpenURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("无效的URL: %v", err)
	}
	return fyne.CurrentApp().OpenURL(parsedURL)
}

func OpenDirectory(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // Linux 或其他类Unix系统
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}
