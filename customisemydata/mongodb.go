package customisemydata

import (
	"dp-cli/config"
	"dp-cli/out"

	"gopkg.in/mgo.v2"
)

func DropMongoData(cfg *config.Config) error {
	if len(cfg.CMD.MongoDBs) == 0 {
		out.Info("no mongo collections specified to drop")
		return nil
	}

	sess, err := mgo.Dial(cfg.CMD.MongoURL)
	if err != nil {
		return err
	}
	defer sess.Close()

	out.InfoF("dropping mongo CMD databases: %+v", cfg.CMD.MongoDBs)
	for _, db := range cfg.CMD.MongoDBs {
		err := sess.DB(db).DropDatabase()
		if err != nil {
			return err
		}
	}

	out.Info("Mongo CMD databases dropped successfully")
	return nil
}
