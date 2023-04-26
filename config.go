package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Folders []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"folders"`
	DBSsh           string `json:"db_ssh"`
	DBDestFolder    string `json:"db_dest_folder"`
	DBContainerName string `json:"db_container_name"`
	PgUser          string `json:"pg_user"`
	PgPassword      string `json:"pg_password"`
	PgHost          string `json:"pg_host"`
	PgPort          string `json:"pg_port"`
	PgDB            string `json:"pg_db"`
	AppDir          string `json:"-"`
}

var config *Config

func LoadConfig(appDir string) {
	config = &Config{AppDir: appDir}
	path := filepath.Join(appDir, "config.json")
	cont, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(cont, config)
	if err != nil {
		log.Fatal(err)
	}
	config.DBContainerName = strings.TrimSpace(config.DBContainerName)
}

func GetConfig() *Config {
	if config == nil {
		log.Fatal("config was not loaded")
	}
	return config
}
