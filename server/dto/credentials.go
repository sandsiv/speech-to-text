package dto

type Credentials struct {
	Credentials  map[string]string `json:"credentials"`
	BucketName   string            `json:"bucketName"`
	EnterpriseId int               `json:"enterpriseId"`
}
