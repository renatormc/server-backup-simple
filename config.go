package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type BackupConfig struct {
	Name    string `json:"name"`
	Folders []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"folders"`
	DBSsh            string   `json:"db_ssh"`
	DBDestFolder     string   `json:"db_dest_folder"`
	DBContainerName  string   `json:"db_container_name"`
	PgUser           string   `json:"pg_user"`
	PgPassword       string   `json:"pg_password"`
	PgHost           string   `json:"pg_host"`
	PgPort           string   `json:"pg_port"`
	PgDB             string   `json:"pg_db"`
	BackupTimes      []string `json:"backup_times"`
	BackupAtStartup  bool     `json:"backup_at_startup"`
	DaysBeforeDelete int64    `json:"days_before_delete"`
	RcloneSync       []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rclone_sync"`
}

type Config struct {
	AppDir string `json:"-"`
}

var config *Config

func LoadConfig(appDir string) *Config {
	config = &Config{AppDir: appDir}
	return config
}

func GetConfig() *Config {
	if config == nil {
		log.Fatal("config was not loaded")
	}
	return config
}

func ReadBackupConfig(name string) (*BackupConfig, error) {
	cf := GetConfig()
	folder := filepath.Join(cf.AppDir, "config")
	c := BackupConfig{}
	path := filepath.Join(folder, fmt.Sprintf("%s.json", name))
	cont, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cont, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func ReadBackupConfigs() []BackupConfig {
	cf := GetConfig()
	folder := filepath.Join(cf.AppDir, "config")
	entries, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal(err)
	}
	ret := []BackupConfig{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		c := BackupConfig{}
		path := filepath.Join(folder, entry.Name())
		cont, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(cont, &c)
		if err != nil {
			log.Fatal(err)
		}
		ret = append(ret, c)
	}
	return ret
}
