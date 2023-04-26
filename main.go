package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/akamensky/argparse"
)

func BackupFolders() {
	cf := GetConfig()
	for _, f := range cf.Folders {
		cmd := exec.Command("rsync", "-avvHPS", "--rsh='ssh'", f.From, f.To)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

func BackupDatabase() {
	cf := GetConfig()
	var cmd *exec.Cmd
	if cf.DBContainerName != "" {
		log.Fatal("not implemented support for docker yet")
		// cmd = exec.Command("ssh", cf.DBSsh, "docker", "exec", "-t", cf.DBContainerName, fmt.Sprintf("PGPASSWORD=%s", cf.PgPassword), "pg_dump", "-d", cf.PgDB, "-U", cf.PgUser, "-p", cf.PgPort, "-h", cf.PgHost, "-O", "-x", "-Ft")
	} else {
		cmd = exec.Command("ssh", cf.DBSsh, fmt.Sprintf("PGPASSWORD=%s", cf.PgPassword), "pg_dump", "-d", cf.PgDB, "-U", cf.PgUser, "-p", cf.PgPort, "-h", cf.PgHost, "-O", "-x", "-Ft")
	}

	now := time.Now()
	fname := fmt.Sprintf("%d_%d_%d-%d_%d_%d.tar", now.Day(), now.Month(), now.Year(), now.Hour(), now.Minute(), now.Second())
	outfile, err := os.Create(filepath.Join(cf.DBDestFolder, fname))
	if err != nil {
		panic(err)
	}
	defer outfile.Close()
	cmd.Stdout = outfile
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	parser := argparse.NewParser("Server backup simple", "App for making backup of database and files from server")

	backupCmd := parser.NewCommand("backup", "Run a backup")

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	LoadConfig(filepath.Dir(ex))
	switch {
	case backupCmd.Happened():
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			BackupDatabase()
			defer wg.Done()
		}()
		go func() {
			BackupFolders()
			defer wg.Done()
		}()
		wg.Wait()
	}
}
