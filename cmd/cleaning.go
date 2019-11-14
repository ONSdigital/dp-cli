package cmd

import (
	"dp-utils/config"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
	"github.com/spf13/cobra"
	"gopkg.in/mgo.v2"
)

var cleanArgs = []string{"cmd"}

func Cleaning(cfg *config.Config) *cobra.Command {
	c := &cobra.Command{
		Use:  "clean",
		Short: "Clean/Delete data from your local environment",
	}
	c.AddCommand(tearDownCustomiseMyData(cfg), clearCollections())
	return c
}

func tearDownCustomiseMyData(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:  "cmd",
		Short: "Drop all CMD data from your local environment",
		Run: func(cmd *cobra.Command, args []string) {
			deleteZebedeeCollections()
			dropMongoData(cfg)
			dropNeo4jData(cfg)
		},
	}
}

func clearCollections() *cobra.Command {
	return &cobra.Command{
		Use:  "collections",
		Short: "Delete all Zebedee collections in your local environment",
		Run: func(cmd *cobra.Command, args []string) {
			deleteZebedeeCollections()
		},
	}
}

func dropMongoData(cfg *config.Config) error {
	if len(cfg.CMD.MongoDBs) == 0 {
		return nil
	}

	sess, err := mgo.Dial(cfg.CMD.MongoURL)
	if err != nil {
		return err
	}
	defer sess.Close()

	color.Cyan("[clean] Dropping mongo CMD databases: %+v", cfg.CMD.MongoDBs)
	for _, db := range cfg.CMD.MongoDBs {
		err := sess.DB(db).DropDatabase()
		if err != nil {
			return err
		}
	}

	color.Cyan("[clean] Mongo CMD databases dropped successfully")
	return nil
}

func dropNeo4jData(cfg *config.Config) error {
	color.Cyan("[clean] Dropping neo4j CMD data")
	pool, err := bolt.NewDriverPool(cfg.CMD.Neo4jURL, 1)
	if err != nil {
		return err
	}

	conn, err := pool.OpenPool()
	if err != nil {
		return err
	}
	defer conn.Close()

	res, err := conn.ExecNeo("MATCH(n) DETACH DELETE n", nil)
	if err != nil {
		return err
	}

	deletions, _ := res.RowsAffected()
	color.Cyan("[clean] Neo4j nodes deleted: %d", deletions)
	return nil
}

func deleteZebedeeCollections() error {
	zebRoot := os.Getenv("zebedee_root")
	collectionsDir := filepath.Join(zebRoot, "/zebedee/collections")

	files, err := filepath.Glob(filepath.Join(collectionsDir, "*"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		color.Cyan("[clean] Zebedee collections deleted successfully")
		return nil
	}

	color.Cyan("[clean] Deleting Zebedee collections: %+v", files)

	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			return err
		}
	}
	color.Cyan("[clean] Zebedee collections deleted successfully")
	return nil
}
