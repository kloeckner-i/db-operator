package gcloud

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

// GetServiceAccount reads file which contains google service account credentials and parse it
func GetServiceAccount() ServiceAccount {
	var serviceaccount ServiceAccount

	credentialPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	credentialFile, err := os.Open(credentialPath)
	if err != nil {
		logrus.Fatal(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer credentialFile.Close()

	credentialValues, _ := ioutil.ReadAll(credentialFile)

	// parse credentials.json file
	json.Unmarshal([]byte(credentialValues), &serviceaccount)

	return serviceaccount
}
