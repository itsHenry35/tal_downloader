package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
)

func IsAndroid() bool {
	return runtime.GOOS == "android"
}

func CleanAndroidTempFolder(app fyne.App) {
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
