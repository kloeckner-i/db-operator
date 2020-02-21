package dbinstance

import (
	"errors"
	"strconv"

	kcidb "github.com/kloeckner-i/db-operator/pkg/utils/database"

	"github.com/sirupsen/logrus"
)

// Generic represents database instance which can be connected by address and port
type Generic struct {
	Host     string
	Port     int32
	Engine   string
	User     string
	Password string
	PublicIP string
}

func makeInterface(in *Generic) (kcidb.Database, error) {
	switch in.Engine {
	case "postgres":
		db := kcidb.Postgres{
			Host:     in.Host,
			Port:     in.Port,
			User:     in.User,
			Password: in.Password,
			Database: "postgres",
		}
		return db, nil
	case "mysql":
		db := kcidb.Mysql{
			Host:     in.Host,
			Port:     in.Port,
			User:     in.User,
			Password: in.Password,
			Database: "mysql",
		}
		return db, nil
	default:
		return nil, errors.New("not supported engine type")
	}
}

func (ins *Generic) exist() error {
	db, err := makeInterface(ins)
	if err != nil {
		logrus.Errorf("can not check if instance exists because of %s", err)
		return err
	}
	err = db.CheckStatus()
	if err != nil {
		logrus.Debug(err)
		return err
	}
	return nil // instance exist
}

func (ins *Generic) create() error {
	return errors.New("creating generic db instance is not yet implimented")
}

func (ins *Generic) update() error {
	logrus.Debug("updating generic db instance is not yet implimented")
	return nil
}

func (ins *Generic) getInfoMap() (map[string]string, error) {
	data := map[string]string{
		"DB_CONN":      ins.Host,
		"DB_PORT":      strconv.FormatInt(int64(ins.Port), 10),
		"DB_PUBLIC_IP": ins.PublicIP,
	}

	return data, nil
}
