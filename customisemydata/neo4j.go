package customisemydata

import (
	"fmt"
	"path/filepath"

	"github.com/ONSdigital/dp-cli/cli"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"

	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
)

func DropNeo4jData(cfg *config.Config) error {
	out.Info("dropping neo4j CMD data")
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
	out.InfoF("neo4j nodes deleted: %d", deletions)
	return nil
}

func ImportGenericHierarchies(cfg *config.Config) error {
	if len(cfg.CMD.Hierarchies) == 0 {
		out.Info("no hierarchies defined in config skipping step")
		return nil
	}

	out.Info(fmt.Sprintf("building generic hierarchies: %+v", cfg.CMD.Hierarchies))

	stopC, progressTicker := cli.GetProgressTicker()
	go progressTicker()

	for _, script := range cfg.CMD.Hierarchies {
		command := fmt.Sprintf("cypher-shell < %s", script)

		if err := cli.ExecCommand(command, filepath.Join(cfg.DPHierarchyBuilderPath, "cypher-scripts")); err != nil {
			stopC <- true
			return err
		}
	}

	stopC <- true

	out.Info("generic hierarchies built successfully")
	return nil
}

func ImportCodeLists(cfg *config.Config) error {
	if len(cfg.CMD.Codelists) == 0 {
		out.Info("no code lists defined in config skipping step")
		return nil
	}

	out.InfoF("importing code lists: %+v", cfg.CMD.Codelists)

	stopC, progressTicker := cli.GetProgressTicker()
	go progressTicker()

	for _, codelist := range cfg.CMD.Codelists {
		command := fmt.Sprintf("./load -q=%s -f=%s", "cypher", codelist)

		if err := cli.ExecCommand(command, filepath.Join(cfg.DPCodeListScriptsPath, "code-list-scripts")); err != nil {
			stopC <- true
			return err
		}
	}
	stopC <- true
	out.Info("code lists imported successfully")
	return nil
}
