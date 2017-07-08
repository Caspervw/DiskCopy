package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const diskIndexDB = "./file-index.db"

var db *sql.DB

func main() {
	bootDatabase()

	diskA := os.Args[1]
	diskB := os.Args[2]

	getFilesFromFolder(true, diskA)
	getFilesFromFolder(false, diskB)

	fmt.Println("--- FINISHED INDEXING ---")
	db.Close()
}

func bootDatabase() {
	//Delete previous DB
	os.Remove(diskIndexDB)

	var err error

	//We're going to store disk A information in a SQLite database
	db, err = sql.Open("sqlite3", diskIndexDB)
	if err != nil {
		log.Fatal("Could not create the SQLite database", err)
	}

	_, err = db.Exec(`
	CREATE TABLE file (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		origin BOOLEAN,
		hash VARCHAR(64),
        path VARCHAR(2048)
    );`)

	if err != nil {
		log.Fatal("DB-TABLE CREATE: fail ", err)
	}
}

func insertFile(origin bool, filePath string) {
	//Not really sure what to do if there is an error, so let's log and quit to be safe. We don't want to mis files
	stmt, err := db.Prepare("INSERT INTO file(origin, hash, path) VALUES (?,?,?)")
	if err != nil {
		log.Fatal("DB-PREPARE: Failed on file: "+filePath, err)
	}

	_, err = stmt.Exec(origin, getMD5FromFile(filePath), filePath)
	if err != nil {
		log.Fatal("DB-INSERT: Failed on file: "+filePath, err)
	}
}

func getFilesFromFolder(origin bool, searchDir string) {
	fmt.Println("Indexing folder: " + searchDir)

	filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			insertFile(origin, path)
		}

		return nil
	})
}

func getMD5FromFile(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("MD5 OPEN: ", filePath, err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal("MD5 IO COPY: ", filePath, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
