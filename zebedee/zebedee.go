package zebedee

import (
	"github.com/ONSdigital/dp-cli/out"
	"os"
	"path/filepath"
)

var zebedeeRoot string

func init() {
	zebedeeRoot = os.Getenv("zebedee_root")
}

func GetZebedeeRoot() string {
	return zebedeeRoot
}

func DeleteCollections() error {
	collectionsDir := filepath.Join(zebedeeRoot, "/zebedee/collections")

	files, err := filepath.Glob(filepath.Join(collectionsDir, "*"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		out.Info("zebedee collections deleted successfully")
		return nil
	}

	out.InfoF("deleting Zebedee collections: %+v", files)

	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			return err
		}
	}
	out.Info("zebedee collections deleted successfully")
	return nil
}
