package models

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/BurntSushi/toml"
)

var (
	configErrors = []string{}
)

type Config struct {
	Operation		string
	Port   			int
	DBPath 			string

	Admin   		[]UserData
}

// Load config settings into given config object
func ReadConfig(conf *Config, configPath string) {
	fileContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln("Configuration file ("+configPath+") not found:", err)
	}
	if md, err := toml.Decode(string(fileContent), &conf); err != nil {
		log.Fatalln(err)
	} else {
		for _, undecoded := range md.Undecoded() {
			errMsg := "[WARN] Undecoded configuration key \"" + undecoded.String() + "\" will not be used."
			configErrors = append(configErrors, errMsg)
			log.Println(errMsg)
		}
	}
}

// Check for config errors and set defaults
func CheckConfig(conf *Config) error {
	if conf.Operation == "" {
		return errors.New("operation must be defined!")
	}
	if len(conf.Admin) == 0 {
		return errors.New("at least one user must be defined!")
	}
	if conf.Port == 0 {
		conf.Port = 80
	}
	if conf.DBPath == "" {
		conf.DBPath = "database.db"
	}
	return nil
}
