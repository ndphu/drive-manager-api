package helper

type KeyDetails struct {
	Type         string `json:"type"`
	ProjectId    string `json:"project_id"`
	PrivateKeyId string `json:"private_key_id"`
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
}
