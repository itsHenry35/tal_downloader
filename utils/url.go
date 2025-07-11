package utils

import (
	"fmt"
	"strings"
)

func ParseVideoUrl(videoURLs []string, message string) (string, error) {
	var videoURL string
	for _, url := range videoURLs {
		if strings.Contains(strings.ToLower(url), ".mp4") {
			videoURL = url
			break
		}
	}

	if videoURL == "" {
		// fallback to m3u8 downloader
		for _, url := range videoURLs {
			if strings.Contains(strings.ToLower(url), ".m3u8") {
				videoURL = url
			}
		}
		if videoURL == "" {
			return "", fmt.Errorf("未找到回放：%s", message)
		}
	}
	return videoURL, nil
}
