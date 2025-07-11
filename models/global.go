package models

type AuthData struct {
	Token  string
	UserID string
}

type Course struct {
	CourseID    string `json:"stdCourseId"`
	TutorID     string `json:"tutorId"`
	CourseName  string `json:"courseName"`
	SubjectName string `json:"subjectName"`
	EndLiveNum  int    `json:"endLiveNum"`
}

type Lecture struct {
	LiveID         int    `json:"liveId"`
	LiveTypeString string `json:"liveTypeString"`
	ClassID        string `json:"stdClassId"`
	SubjectID      string `json:"stdSubject"`
	LecturerID     string `json:"lecturerId"`
}

type StudentAccount struct {
	PuUID                 int    `json:"pu_uid"`
	Nickname              string `json:"nickname"`
	IsCurrentLoginAccount bool   `json:"isCurrentLoginAccount"`
}
