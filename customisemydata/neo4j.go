package customisemydata

import (
	"dp-cli/config"
	"dp-cli/out"
	"dp-cli/utils"
	"fmt"

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

func ImportGenericHierarchies(hierarchyBuilderPath string, cfg *config.Config) error {
	if len(cfg.CMD.Hierarchies) == 0 {
		out.Info("no hierarchies defined in config skipping step")
		return nil
	}

	out.Info(fmt.Sprintf("building generic hierarchies: %+v", cfg.CMD.Hierarchies))

	stopC, progressTicker := utils.GetProgressTicker()
	go progressTicker()

	for _, script := range cfg.CMD.Hierarchies {
		command := fmt.Sprintf("cypher-shell < %s/%s", hierarchyBuilderPath, script)

		if err := utils.ExecCommand(command, ""); err != nil {
			stopC <- true
			return err
		}
	}

	stopC <- true

	out.Info("generic hierarchies built successfully")
	return nil
}

func ImportCodeLists(codeListScriptsPath string, cfg *config.Config) error {
	if len(cfg.CMD.Codelists) == 0 {
		out.Info("no code lists defined in config skipping step")
		return nil
	}

	out.InfoF("importing code lists: %+v", cfg.CMD.Codelists)

	stopC, progressTicker := utils.GetProgressTicker()
	go progressTicker()

	for _, codelist := range cfg.CMD.Codelists {
		command := fmt.Sprintf("./load -q=%s -f=%s", "cypher", codelist)

		if err := utils.ExecCommand(command, codeListScriptsPath); err != nil {
			stopC <- true
			return err
		}
	}
	stopC <- true
	out.Info("code lists imported successfully")
	return nil
}
