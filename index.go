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

const (
	diskIndexDB = "./file-index.db"
	breakPoint  = 10000
)

var db *sql.DB

func main() {
	//init
	os.Remove(diskIndexDB)
	os.Create("indexer.txt")
	os.Create("copy.txt")

	bootDatabase()

	diskA := os.Args[1]
	diskB := os.Args[2]
	dest := os.Args[3]

	getFilesFromFolder(true, diskA)
	getFilesFromFolder(false, diskB)

	fmt.Println("--- FINISHED INDEXING ---")
	fmt.Println("--- STARTING COPY ---")

	files := findMissingFiles()
	copyFiles(dest, files)

	db.Close()

	fmt.Println("--- FINISHED ---")

}

func bootDatabase() {
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
		redoLog("indexer", filePath)
		return
	}

	md5, err := getMD5FromFile(filePath)
	if err != nil {
		redoLog("indexer", filePath)
		return
	}

	_, err = stmt.Exec(origin, md5, filePath)
	if err != nil {
		redoLog("indexer", filePath)
		return
	}
}

func getFilesFromFolder(origin bool, searchDir string) {
	fmt.Println("Indexing folder: " + searchDir)

	var count int64 = 0

	filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			count++
			if count%breakPoint == 0 {
				fmt.Println(count)
			}
			insertFile(origin, path)
		}

		return nil
	})
}

func getMD5FromFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func findMissingFiles() []string {
	//We left join to get the missing files and retrieve only the ones from the source
	missingQuery := `
	SELECT source.path,
		destination.id AS missing
	FROM   FILE source
		LEFT JOIN FILE destination
				ON source.hash = destination.hash
					AND source.origin = 1
					AND destination.origin = 0
	WHERE  missing IS NULL
		AND source.origin = 1`

	rows, err := db.Query(missingQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	files := []string{}
	for rows.Next() {
		var path string
		var weDoNotCare interface{}
		err = rows.Scan(&path, &weDoNotCare)

		if err != nil {
			log.Fatal(err)
		}

		files = append(files, path)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return files

}

func copyFiles(dir string, files []string) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatal("COPY INIT: Could not create copy-directory")
	}

	lengthFiles := len(files)
	count := 0
	for _, file := range files {
		count++
		percentage := int((count / lengthFiles) * 100)

		if percentage%2 == 0 {
			fmt.Println(fmt.Sprintf("%d%%", percentage))
		}

		err = copyFile(file, dir+file)

		if err != nil {
			redoLog("copy", file)
		}
	}
}

func copyFile(src string, dst string) (err error) {
	//Create output dir
	dir := filepath.Dir(dst)
	os.MkdirAll(dir, os.ModePerm)

	//Do the copy
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

//Dirty function to write the filepaths that crashedd to a file for after-processing
func redoLog(tag string, path string) {
	f, _ := os.OpenFile(tag+".txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	f.WriteString(path + "\n")
	f.Close()
}
