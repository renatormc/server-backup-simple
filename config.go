package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type BackupConfig struct {
	Name    string `toml:"name"`
	Folders []struct {
		From string `toml:"from"`
		To   string `toml:"to"`
	} `toml:"folders"`
	DBSsh            string   `toml:"db_ssh"`
	DBDestFolder     string   `toml:"db_dest_folder"`
	DBContainerName  string   `toml:"db_container_name"`
	PgUser           string   `toml:"pg_user"`
	PgPassword       string   `toml:"pg_password"`
	PgHost           string   `toml:"pg_host"`
	PgPort           string   `toml:"pg_port"`
	PgDB             string   `toml:"pg_db"`
	BackupTimes      []string `toml:"backup_times"`
	BackupAtStartup  bool     `toml:"backup_at_startup"`
	DaysBeforeDelete int64    `toml:"days_before_delete"`
	RcloneSync       []struct {
		From string `toml:"from"`
		To   string `toml:"to"`
	} `toml:"rclone_sync"`
}

type Config struct {
	AppDir  string
	LogFile string
}

var config *Config

func LoadConfig(appDir string) *Config {
	config = &Config{AppDir: appDir, LogFile: filepath.Join(appDir, "log.txt")}
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
	path := filepath.Join(folder, fmt.Sprintf("%s.toml", name))
	_, err := toml.DecodeFile(path, &c)
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
		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		c := BackupConfig{}
		path := filepath.Join(folder, entry.Name())

		_, err := toml.DecodeFile(path, &c)
		if err != nil {
			log.Fatal(err)
		}
		ret = append(ret, c)
	}
	return ret
}
