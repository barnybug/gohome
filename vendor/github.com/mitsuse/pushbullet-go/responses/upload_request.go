package responses

type Upload struct {
	Data      *UploadData `json:"data"`
	FileName  string      `json:"file_name"`
	FileType  string      `json:"file_type"`
	FileUrl   string      `json:"file_url"`
	UploadUrl string      `json:"upload_url"`
}

type UploadData struct {
	Acl            string `json:"acl"`
	AwsAccessKeyId string `json:"awsaccesskeyid"`
	ContentType    string `json:"content-type"`
	Key            string `json:"key"`
	Policy         string `json:"policy"`
	Signature      string `json:"signature"`
}
