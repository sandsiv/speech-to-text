package dto

type Text struct {
	Uuid       string `json:"uuid"`
	PathToFile string `json:"path_to_file"`
	Text       string `json:"text"`
	Duration   int32  `json:"duration"`
	Language   string `json:"language"`
}
