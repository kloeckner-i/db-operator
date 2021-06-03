package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sdomino/scribble"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

type apiService struct {
	DB       *scribble.Driver
	project  string
	instance string
	user     string
	host     string
}

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

func newapiService() *apiService {
	db, err := scribble.New("data", nil)
	if err != nil {
		logrus.Errorf("can not set up scribble db", err)
	}

	return &apiService{
		DB: db,
	}
}

func healthCheckRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("healthy")
}

func getInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("get instance called - %s %s", vars["project"], vars["instance"])

	w.Header().Set("Content-Type", "application/json")
	if r.Body != http.NoBody {
		err := fmt.Errorf("get instance invalid body nil")
		logrus.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	s := newapiService()
	s.project = vars["project"]
	s.instance = vars["instance"]

	dbin, err := s.readInstance()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode("instanceDoesNotExist")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dbin)
}

func createInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("create instance called - %s %s", vars["project"], "-")
	w.Header().Set("Content-Type", "application/json")

	dbin := sqladmin.DatabaseInstance{}
	err := json.NewDecoder(r.Body).Decode(&dbin)
	if err != nil {
		logrus.Errorf("create instance invalid body - %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	s := newapiService()
	s.project = vars["project"]
	s.instance = dbin.Name

	_, err = s.readInstance()
	if err == nil {
		logrus.Infof("instance already exist - %s %s", vars["project"], dbin.Name)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode("instanceAlreadyExists")
		return
	}

	logrus.Infof("creating instance - %s %s", vars["project"], dbin.Name)
	dbin.State = "PENDING_CREATE"
	err = s.writeInstance(dbin)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	defer s.setInstanceRunning()

	logrus.Infof("instance created - %s %s", vars["project"], dbin.Name)
	op := successOperation(s.project, s.instance, "CREATE", w.Header().Clone())

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(op)
}

func patchInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("patch instance called - %s %s", vars["project"], vars["instance"])
	w.Header().Set("Content-Type", "application/json")

	s := newapiService()
	s.project = vars["project"]
	s.instance = vars["instance"]
	dbin, err := s.readInstance()
	if err != nil {
		logrus.Infof("instance does not exist - %s %s", s.project, s.instance)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&dbin)
	if err != nil {
		// nothing changed just ok
		w.WriteHeader(http.StatusOK)
		return
	}

	logrus.Infof("patching instance - %s %s", vars["project"], dbin.Name)
	dbin.State = "PENDING_UPDATE"
	err = s.writeInstance(dbin)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	defer s.setInstanceRunning()

	logrus.Infof("instance updated - %s %s", vars["project"], dbin.Name)
	op := successOperation(s.project, s.instance, "UPDATE", w.Header().Clone())

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(op)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Infof("update user called  - %s %s", vars["project"], vars["instance"])
	w.Header().Set("Content-Type", "application/json")

	hostList, ok := r.URL.Query()["host"]
	if !ok || hostList[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("Missing parameter: host.")
		return
	}
	nameList, ok := r.URL.Query()["name"]
	if !ok || nameList[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("Missing parameter: user.")
		return
	}

	sqlUser := sqladmin.User{}
	err := json.NewDecoder(r.Body).Decode(&sqlUser)
	if err != nil {
		logrus.Errorf("user update invalid body - %s", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	logrus.Infof("Request body: %v", sqlUser)

	s := newapiService()
	s.project = vars["project"]
	s.instance = vars["instance"]
	s.user = nameList[0]
	s.host = hostList[0]
	logrus.Infof("updating user - %s %s %s %s", s.project, s.instance, s.user, s.host)

	err = s.writeUser(sqlUser)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return
	}

	op := successOperation(s.project, s.instance, "UPDATE", w.Header().Clone())
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(op)
}

func (s *apiService) readInstance() (sqladmin.DatabaseInstance, error) {
	logrus.Infof("db reading %s - %s", s.project, s.instance)
	dbin := sqladmin.DatabaseInstance{}
	if err := s.DB.Read(s.project, s.instance, &dbin); err != nil {
		logrus.Errorf("failed to read instance to db", err)
		return dbin, err
	}

	return dbin, nil
}

func (s *apiService) writeInstance(dbin sqladmin.DatabaseInstance) error {
	logrus.Infof("db writing %s - %s", s.project, s.instance)
	err := s.DB.Write(s.project, s.instance, dbin)
	if err != nil {
		logrus.Errorf("failed to write instance to db", err)
		return err
	}
	return nil
}

func (s *apiService) setInstanceRunning() {
	const delay = 5
	time.Sleep(delay * time.Second)

	dbin, err := s.readInstance()
	if err != nil {
		logrus.Fatal("can not get instance state")
	}

	dbin.State = "RUNNABLE"
	err = s.writeInstance(dbin)
	if err != nil {
		logrus.Fatal("can not update instance state")
	}
	logrus.Infof("instance is running - %s %s", s.project, s.instance)
}

func (s *apiService) writeUser(user sqladmin.User) error {
	rc := fmt.Sprintf("%s-%s-%s", s.instance, s.user, s.host)
	err := s.DB.Write(s.project, rc, user)
	if err != nil {
		logrus.Errorf("failed to write user", err)
		return err
	}
	return nil
}

func successOperation(project, instance, opType string, header http.Header) sqladmin.Operation {
	return sqladmin.Operation{
		Kind:          "sql#operation",
		Status:        "PENDING",
		OperationType: opType,
		TargetId:      instance,
		TargetProject: project,
		ServerResponse: googleapi.ServerResponse{
			Header:         header,
			HTTPStatusCode: http.StatusOK,
		},
	}
}
