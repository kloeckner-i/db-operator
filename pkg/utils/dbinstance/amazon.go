package dbinstance

import (
	"errors"
	"strconv"

	kcidb "github.com/kloeckner-i/db-operator/pkg/utils/database"

	"github.com/sirupsen/logrus"
)

// Amazon represents database instance which can be connected by address and port
type Amazon struct {
	Host               string
	Port               uint16
	Engine             string
	User               string
	Password           string
	PublicIP           string
	ServiceAccountName string
}

func (in *Amazon) exist() error {
	if "postgres" != in.Engine {
		return errors.New("not supported engine type")
	}
	db := kcidb.Postgres{
		Host:     in.Host,
		Port:     in.Port,
		User:     in.User,
		Password: in.Password,
		Database: "postgres",
	}
	err := db.CheckStatus()
	if err != nil {
		logrus.Debug(err)
		return err
	}
	return nil // instance exist
}

func (ins *Amazon) create() error {
	return errors.New("creating amazon db instance is not yet implimented")
}

func (ins *Amazon) update() error {
	logrus.Debug("updating amazon db instance is not yet implimented")
	return nil
}

func (ins *Amazon) getInfoMap() (map[string]string, error) {
	data := map[string]string{
		"DB_CONN":      ins.Host,
		"DB_PORT":      strconv.FormatInt(int64(ins.Port), 10),
		"DB_PUBLIC_IP": ins.PublicIP,
	}

	return data, nil
}
