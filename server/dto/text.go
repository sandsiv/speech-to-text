package dto

type Text struct {
	Uuid             string `json:"uuid"`
	FileUrl          string `json:"fileUrl"`
	Text             string `json:"text"`
	Duration         int32  `json:"duration"`
	Language         string `json:"language"`
	EnterpriseId     int    `json:"enterpriseId"`
	Error            error  `json:"-"`
	RecognitionError error  `json:"-"`
	Link             string `json:"-"`
	FilePath         string `json:"-"`
}
