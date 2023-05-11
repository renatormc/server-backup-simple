package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/akamensky/argparse"
	"github.com/jasonlvhit/gocron"
)

const TIME_LAYOUT = "2006-01-02 15_04_05"

func BackupFolders(c BackupConfig) {
	for _, f := range c.Folders {
		args := []string{"rsync", "-avvHPS", "--rsh='ssh'", f.From, f.To}
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("wsl", args...)
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
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
		if c.DBSsh == "" {
			cmd = exec.Command("pg_dump", "-d", c.PgDB, "-U", c.PgUser, "-p", c.PgPort, "-h", c.PgHost, "-O", "-x", "-Ft")
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", c.PgPassword))
		} else {
			cmd = exec.Command("ssh", c.DBSsh, fmt.Sprintf("PGPASSWORD=%s", c.PgPassword), "pg_dump", "-d", c.PgDB, "-U", c.PgUser, "-p", c.PgPort, "-h", c.PgHost, "-O", "-x", "-Ft")
		}

	}

	now := time.Now()
	fname := fmt.Sprintf("%s.tar", now.Format(TIME_LAYOUT))
	outfile, err := os.Create(filepath.Join(c.DBDestFolder, fname))
	if err != nil {
		log.Println(err)
	}
	defer outfile.Close()
	cmd.Stdout = outfile
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}

func BackupAll(c BackupConfig) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		log.Printf("Iniciando backup do banco de %q\n", c.Name)
		BackupDatabase(c)
		log.Printf("Backup do banco de %q finalizado.\n", c.Name)
		defer wg.Done()
	}()
	go func() {
		log.Printf("Iniciando sincronização das pastas de %q\n", c.Name)
		BackupFolders(c)
		log.Printf("Sincronização das pastas de %q finalizada.\n", c.Name)
		defer wg.Done()
	}()
	wg.Wait()
	RcloneSync(c)
}

func DeleteOld(c BackupConfig) {
	log.Printf("Deletando backups antigos de %q\n", c.Name)
	entries, err := os.ReadDir(c.DBDestFolder)
	if err != nil {
		log.Println(err)
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

func RcloneSync(c BackupConfig) {
	log.Printf("Sincronizando rclone de %q\n", c.Name)
	for _, item := range c.RcloneSync {
		cmd := exec.Command("rclone", "sync", item.From, item.To)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	}

}

func ReadTail(fileName string, n int) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= n {
		return strings.Join(lines, "\n"), nil
	}
	return strings.Join(lines[len(lines)-n:], "\n"), nil
}

func main() {
	parser := argparse.NewParser("Server backup simple", "App for making backup of database and files from server")
	logToFile := parser.Flag("l", "logfile", &argparse.Options{Default: false, Help: "Log to file instead of console"})

	backupCmd := parser.NewCommand("backup", "Run a backup")
	configName := backupCmd.StringPositional(&argparse.Options{Help: "Log to file instead of console"})

	schedulerCmd := parser.NewCommand("scheduler", "Start scheduler")

	deleteOldCmd := parser.NewCommand("delete-old", "Delete old files")

	logCmd := parser.NewCommand("log", "Print log")
	nLines := logCmd.Int("n", "lines", &argparse.Options{Default: 10, Help: "Number of lines of the tail"})

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
		f, err := os.OpenFile(cf.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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
	case logCmd.Happened():
		text, err := ReadTail(cf.LogFile, *nLines)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(text)
	}
}
