package config

import "fyne.io/fyne/v2"

const (
	// API Base URLs
	PassportAPIBase       = "https://passport.100tal.com"
	CourseAPIBase_Ledu    = "https://course-api-online.saasp.vdyoo.com"
	CourseAPIBase_XES     = "https://course-api-online.speiyou.com"
	ClassroomAPIBase_Ledu = "https://classroom-api-online.saasp.vdyoo.com"
	ClassroomAPIBase_XES  = "https://classroom-api-online.speiyou.com"

	// Names
	PlatformName_Ledu = "乐读"
	PlatformName_XES  = "学而思培优"

	// Client Configuration
	ClientID_Ledu = "523601"
	ClientID_XES  = "123601"
	DeviceID      = "TAL"
	Terminal      = "pc"
	Version       = "3.21.0.84"
	ResVer        = "1.0.6"
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36"

	// Download Configuration
	MaxConcurrentDownloads = 12
	ThreadCount            = 16
)

var (
	DefaultWindowSize = fyne.NewSize(800, 600)
	CourseAPIBase     = CourseAPIBase_Ledu    // Default to Ledu, can be changed to XES
	ClassroomAPIBase  = ClassroomAPIBase_Ledu // Default to Ledu, can be changed to XES
	ClientID          = ClientID_Ledu         // Default to Ledu, can be changed to XES
	PlatformName      = PlatformName_Ledu     // Default to Ledu, can be changed to XES
)

func SetPlatform(platform string) {
	switch platform {
	case "ledu":
		CourseAPIBase = CourseAPIBase_Ledu
		ClassroomAPIBase = ClassroomAPIBase_Ledu
		ClientID = ClientID_Ledu
		PlatformName = PlatformName_Ledu
	case "xes":
		CourseAPIBase = CourseAPIBase_XES
		ClassroomAPIBase = ClassroomAPIBase_XES
		ClientID = ClientID_XES
		PlatformName = PlatformName_XES
	default:
		panic("Unsupported platform: " + platform)
	}
}
