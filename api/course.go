package api

import (
	"encoding/json"
	"fmt"

	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/models"
	"github.com/itsHenry35/tal_downloader/utils"
)

// GetCourseList retrieves the list of courses, paginated fetch until empty result
func (c *Client) GetCourseList() ([]*models.Course, error) {
	var allCourses []*models.Course
	page := 1
	perPage := 10

	for {
		coursesURL := fmt.Sprintf("%s/course/v1/student/course/list?stuId=%s&courseStatus=0&stdSubject=&page=%d&perPage=%d&order=desc",
			config.CourseAPIBase, c.userID, page, perPage)

		resp, err := c.doRequest("GET", coursesURL, nil, nil, false)
		if err != nil {
			return nil, err
		}

		var courses []*models.Course
		if err := json.NewDecoder(resp.Body).Decode(&courses); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(courses) == 0 {
			break // no more data
		}

		allCourses = append(allCourses, courses...)
		page++
	}

	return allCourses, nil
}

// GetLectures retrieves lectures for a course, paginated fetch until empty result
func (c *Client) GetLectures(courseID string) ([]*models.Lecture, error) {
	var allLectures []*models.Lecture
	page := 1
	perPage := 10

	for {
		lecturesURL := fmt.Sprintf("%s/course/v1/student/course/user-live-list?stuId=%s&stdCourseId=%s&type=1&needPage=1&page=%d&perPage=%d&order=asc",
			config.CourseAPIBase, c.userID, courseID, page, perPage)

		resp, err := c.doRequest("GET", lecturesURL, nil, nil, false)
		if err != nil {
			return nil, err
		}

		var lectures []*models.Lecture
		if err := json.NewDecoder(resp.Body).Decode(&lectures); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(lectures) == 0 {
			break // no more data
		}

		allLectures = append(allLectures, lectures...)
		page++
	}

	return allLectures, nil
}

// GetVideoURL retrieves the download URL for a video
func (c *Client) GetVideoURL(lecture *models.Lecture, courseID, tutorID string) (string, error) {
	headers := map[string]string{
		"Host":          "classroom-api-online.saasp.vdyoo.com",
		"lecturerId":    lecture.LecturerID,
		"stdSubject":    lecture.SubjectID,
		"tutorId":       tutorID,
		"stdCourseId":   courseID,
		"liveId":        fmt.Sprintf("%d", lecture.LiveID),
		"liveType":      lecture.LiveTypeString,
		"stdClassId":    lecture.ClassID,
		"expireTime":    "0",
		"appClientType": "xes",
	}

	switch lecture.LiveTypeString {
	case "SMALL_GROUPS_V2_MODE", "COMBINE_SMALL_CLASS_MODE", "SMALL_CLASS_MODE", "GENERAL_V2_MODE":
		url := fmt.Sprintf("%s/playback/v1/video/init", config.ClassroomAPIBase)
		resp, err := c.doRequest("GET", url, nil, headers, false)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result models.VideoUrlResponse

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		return utils.ParseVideoUrl(result.VideoURLs, result.Message)

	case "RECORD_MODE", "ONLINE_REAL_RECORD":
		url := fmt.Sprintf("%s/classroom-ai/record/v1/resources", config.ClassroomAPIBase)
		resp, err := c.doRequest("GET", url, nil, headers, false)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result models.RecordModeVideoUrlResponse

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		var videoURL string

		for _, urls := range result.Definitions {
			if len(urls) > 0 {
				videoURL = urls[len(urls)-1]
				break
			}
		}

		if videoURL == "" {
			return "", fmt.Errorf("未找到回放：%s", result.Message)
		}
		return videoURL, nil

	default:
		return "", fmt.Errorf("unsupported live type: %s", lecture.LiveTypeString)
	}

}
