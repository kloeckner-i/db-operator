package gcloud

// ServiceAccount is google service account
type ServiceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientID    string `json:"client_id"`
	ClientEmail string `json:"client_email"`
}
