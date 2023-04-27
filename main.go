package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/akamensky/argparse"
	"github.com/jasonlvhit/gocron"
)

const TIME_LAYOUT = "2006-01-02 15_04_05"

func BackupFolders(c BackupConfig) {
	for _, f := range c.Folders {
		cmd := exec.Command("rsync", "-avvHPS", "--rsh='ssh'", f.From, f.To)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	}
}

func BackupDatabase(c BackupConfig) {
	var cmd *exec.Cmd
	if c.DBContainerName != "" {
		log.Fatal("not implemented support for docker yet")
		// cmd = exec.Command("ssh", cf.DBSsh, "docker", "exec", "-t", cf.DBContainerName, fmt.Sprintf("PGPASSWORD=%s", cf.PgPassword), "pg_dump", "-d", cf.PgDB, "-U", cf.PgUser, "-p", cf.PgPort, "-h", cf.PgHost, "-O", "-x", "-Ft")
	} else {
		cmd = exec.Command("ssh", c.DBSsh, fmt.Sprintf("PGPASSWORD=%s", c.PgPassword), "pg_dump", "-d", c.PgDB, "-U", c.PgUser, "-p", c.PgPort, "-h", c.PgHost, "-O", "-x", "-Ft")
	}

	now := time.Now()
	// fname := fmt.Sprintf("%d_%d_%d-%d_%d_%d.tar", now.Day(), now.Month(), now.Year(), now.Hour(), now.Minute(), now.Second())
	fname := fmt.Sprintf("%s.tar", now.Format(TIME_LAYOUT))
	outfile, err := os.Create(filepath.Join(c.DBDestFolder, fname))
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

func BackupAll(c BackupConfig) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		log.Printf("Iniciando backup do banco de %q\n", c.Name)
		BackupDatabase(c)
		defer wg.Done()
	}()
	go func() {
		log.Printf("Iniciando sincronização das pastas de %q\n", c.Name)
		BackupFolders(c)
		defer wg.Done()
	}()
	wg.Wait()
}

func DeleteOld(c BackupConfig) {
	log.Printf("Deletando backups antigos de %q\n", c.Name)
	entries, err := os.ReadDir(c.DBDestFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar") {
			t, err := time.Parse(TIME_LAYOUT, entry.Name()[:len(entry.Name())-4])
			now := time.Now()
			then := now.Add(-1 * time.Duration(24*c.DaysBeforeDelete) * time.Hour)
			if err != nil || t.Before(then) {
				if err := os.Remove(filepath.Join(c.DBDestFolder, entry.Name())); err != nil {
					log.Printf("not possible to delete %q\n", entry.Name())
				}

			}
			fmt.Println(t)
		}
	}
}

func main() {
	parser := argparse.NewParser("Server backup simple", "App for making backup of database and files from server")
	logToFile := parser.Flag("l", "logfile", &argparse.Options{Default: false, Help: "Log to file instead of console"})

	backupCmd := parser.NewCommand("backup", "Run a backup")
	configName := backupCmd.StringPositional(&argparse.Options{Help: "Log to file instead of console"})

	schedulerCmd := parser.NewCommand("scheduler", "Start scheduler")

	deleteOldCmd := parser.NewCommand("delete-old", "Delete old files")

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	cf := LoadConfig(filepath.Dir(ex))
	if *logToFile {
		f, err := os.OpenFile(filepath.Join(cf.AppDir, "log.txt"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		log.SetOutput(f)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	switch {
	case backupCmd.Happened():
		c, err := ReadBackupConfig(*configName)
		if err != nil {
			log.Fatal(err)
		}
		BackupAll(*c)
	case schedulerCmd.Happened():
		for _, c := range ReadBackupConfigs() {
			if c.BackupAtStartup {
				DeleteOld(c)
				BackupAll(c)
			}

			for _, t := range c.BackupTimes {
				err = gocron.Every(1).Day().At(t).Do(func() {
					DeleteOld(c)
					BackupAll(c)
				})
				if err != nil {
					log.Fatal(err.Error())
				}
			}
		}

		log.Println("Starting scheduler")
		<-gocron.Start()
	case deleteOldCmd.Happened():
		for _, c := range ReadBackupConfigs() {
			DeleteOld(c)
		}

	}
}
