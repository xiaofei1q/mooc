package config

type Config struct {
	Global Global `json:"global"`
	Users  []User `json:"users"`
}
type Global struct {
	Server string `json:"server"`
	Limit  int    `json:"limit"`
}
type User struct {
	BaseURL    string   `json:"base_url"`
	SchoolID   int      `json:"school_id"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	CourseNames []string `json:"course_names"`
}

var Conf Config

const VERSION = "v1.3.3plus"
