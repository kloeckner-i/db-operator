package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func main() {
	//set loglevel
	setLogLevel()

	//initialize router
	router := mux.NewRouter()

	//endpoint
	router.HandleFunc("/health", healthCheckRequestHandler).Methods("GET")
	router.HandleFunc("/sql/v1beta4/projects/{project}/instances/{instance}", getInstanceHandler).Methods("GET")
	router.HandleFunc("/sql/v1beta4/projects/{project}/instances", createInstanceHandler).Methods("POST")
	router.HandleFunc("/sql/v1beta4/projects/{project}/instances/{instance}", patchInstanceHandler).Methods("PATCH")
	router.HandleFunc("/sql/v1beta4/projects/{project}/instances/{instance}/users", updateUserHandler).Methods("PUT")

	logrus.Info("Listen :8080")
	logrus.Fatal(http.ListenAndServe(":8080", router))
}

func setLogLevel() {
	level := os.Getenv("LOG_LEVEL")
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.Printf("can not parse loglevel")
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)
}

func healthCheckRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("healthy")
}

func getInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("%s %s - get instance called", vars["project"], vars["instance"])

	w.Header().Set("Content-Type", "application/json")
	if r.Body != http.NoBody {
		logrus.Error("get instance invalid body nil")
		w.WriteHeader(http.StatusBadRequest)

	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("sent")

}

func createInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("%s %s - create instance called", vars["project"], "-")

	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&sqladmin.DatabaseInstance{})
	if err != nil {
		logrus.Errorf("create instance invalid body - %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("sent")
	}
}

func patchInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("%s %s - patch instance called", vars["project"], vars["instance"])

	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&sqladmin.DatabaseInstance{})
	if err != nil {
		logrus.Errorf("create instance invalid body - %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("sent")
	}
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("%s %s - update user called", vars["project"], vars["instance"])

	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&sqladmin.User{})
	if err != nil {
		logrus.Errorf("create instance invalid body - %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("sent")
	}
}
