package utils

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func ReadConfig(cliCtx *cli.Context, o interface{}) error {
	configFilePath := cliCtx.GlobalString(ConfigFileFlag.Name)

	if configFilePath != "" {
		err := readYamlConfig(configFilePath, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func readFile(path string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func readYamlConfig(path string, o interface{}) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Fatal("Path ", path, " does not exist")
	}
	b, err := readFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, o)
	if err != nil {
		log.Fatalf("unable to parse file with error %#v", err)
	}

	return nil
}
