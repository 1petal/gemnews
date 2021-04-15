package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	// _ "modernc.org/sqlite"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

func dbinit(dbpath string, debug bool) *sql.DB {

	if !fileExists(dbpath) {
		createDBfile(dbpath)
		sqliteDatabase, _ := sql.Open("sqlite3", dbpath)
		newsTablesInit(sqliteDatabase, debug)
		sqliteDatabase.Close()
	}

	sqliteDatabase, _ := sql.Open("sqlite3", dbpath) // Open the created SQLite File

	// defer sqliteDatabase.Close()                     // Defer Closing the database in main
	//updateHashes(sqliteDatabase)

	return sqliteDatabase
}

func createDBfile(path string) {
	log.Println("Creating sqlite-database.db at", path)

	file, err := os.Create(path) // Create SQLite file
	if err != nil {
		log.Fatal(err.Error())

		// Running btrfs, need to disable COW
		chattrBin := which("chattr")
		if _, err := os.Stat(path); err == nil {
			cmd := exec.Command(chattrBin, "+C", path)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			if err = cmd.Run(); err != nil {
				//return fmt.Printf("%s +C failed: %s. Err: %v", chattrBin, stderr.String(), err)
				fmt.Printf("%s +C failed: %s. Err: %v", chattrBin, stderr.String(), err)
			}
		}

	}
	file.Close()
}

func newsTablesInit(db *sql.DB, debug bool) {

	createNewsArticleTableSQL := `CREATE TABLE if not exists newsarticle (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"hash" integer KEY,
		"title" TEXT,
		"url" TEXT,
		"domain" TEXT,
		"date", int
		);` // SQL Statement for Create Table

	statement, err := db.Prepare(createNewsArticleTableSQL) // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec() // Execute SQL Statements

	if debug {
		log.Println("newsarticle table init exec")
	}

	createArticlesTableSQL := `CREATE TABLE if not exists articles (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"hash" integer KEY,
		"title" TEXT,
		"url" TEXT,
		"topimage" TEXT,
		"domain" TEXT,
		"date" integer, 
		"keywords" TEXT, 
		"description" TEXT,
		"raw" integer, 
		"content" BLOB
		);` // SQL Statement for Create Table

	statement, err = db.Prepare(createArticlesTableSQL) // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec() // Execute SQL Statements

	if debug {
		log.Println("newsarticle table init exec")
	}

}

// We are passing db reference connection from main to our method with other parameters
func articleInsertIndex(db *sql.DB, title, url, domain string, date uint64, hash uint32) {

	insertIp2cSQL := `INSERT INTO newsarticle(title, url, domain, date, hash) VALUES (?, ?, ?, ?, ?)`
	statement, err := db.Prepare(insertIp2cSQL) // Prepare statement.
	// This is good to avoid SQL injections
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = statement.Exec(title, url, domain, date, hash)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func articleInsertContent(db *sql.DB, title, url, domain, topimage, keywords, description, content string, raw int) {

	insertArticleSQL := `INSERT INTO articles(title, url, domain, topimage, keywords, description, hash, content, raw, date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	statement, err := db.Prepare(insertArticleSQL) // Prepare statement.
	// This is good to avoid SQL injections
	if err != nil {
		log.Fatalln("db.Prep", err.Error())
	}
	urlhash := urlhash(url)

	_, err = statement.Exec(title, url, domain, topimage, keywords, description, urlhash, zip(content), raw, getEpochTime())
	if err != nil {
		log.Fatalln("db.Exec", err.Error())
	}
}

func articleInIndex(db *sql.DB, hash uint32, debug bool) bool {

	row := db.QueryRow("select title from newsarticle where hash= ?", hash)

	temp := ""
	row.Scan(&temp)
	if temp != "" {
		if debug {
			fmt.Println("articleInIndex:hash", hash, "found:", temp)
		}
		return true
	}
	if debug {
		fmt.Println("articleInIndex:hash", hash, "not found:", temp)
	}
	return false
}

func articleNoContent(db *sql.DB, URL string, debug bool) bool {

	hash := urlhash(URL)
	row := db.QueryRow("select url from articles where hash= ?", hash)

	temp := ""
	row.Scan(&temp)
	return temp == "" //False if content, True if no content
}

func updateHashes(db *sql.DB) { //this shit didn't work

	row, err := db.Query("SELECT id,title,URL,domain,hash,date FROM newsarticle order by id DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	//title, url, domain string, date uint64, hash uint32

	var (
		id     int
		title  string
		URL    string
		domain string
		hash   uint32
		date   uint64
	)

	/* for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&id, &title, &URL, &domain, &hash, &date)
		fmt.Println("article:", date, id, title, URL, domain, hash)

	} */

	for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&id, &title, &URL, &domain, &hash, &date)
		articlehash := smallhash(fmt.Sprintf("%s%s", URL, domain))
		if articlehash != hash {
			trashSQL, err := db.Prepare("update newsarticle set hash=? where id=?")
			if err != nil {
				fmt.Println(err)
			}

			tx, err := db.Begin()
			if err != nil {
				fmt.Println(err)
			}
			_, err = tx.Stmt(trashSQL).Exec(articlehash, id)
			if err != nil {
				fmt.Println("doing rollback")
				tx.Rollback()
			} else {
				fmt.Println("updateing", id, domain, date, hash, "to", articlehash)
				tx.Commit()
			}
		} else {
			fmt.Printf(".")
		}
	}
}

func displayArticleAll(db *sql.DB) {
	row, err := db.Query("SELECT id,title,URL,domain,hash,date FROM newsarticle order by date DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	//title, url, domain string, date uint64, hash uint32

	var (
		id     int
		title  string
		URL    string
		domain string
		hash   uint32
		date   uint64
	)

	for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&id, &title, &URL, &domain, &hash, &date)
		fmt.Println("article:", date, id, title, URL, domain, hash)

	}
}

func displayArticleHours(db *sql.DB, oldhours int) {

	backHours := time.Hour * time.Duration(-oldhours)

	loc, _ := time.LoadLocation("UTC")
	tThen := time.Now().In(loc).Add(backHours).Unix()

	fmt.Println("looking for articles newer than", tThen)

	row, err := db.Query("SELECT id,title,URL,domain,hash,date FROM newsarticle WHERE date > ? order by date DESC", tThen)
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	//title, url, domain string, date uint64, hash uint32

	var (
		id     int
		title  string
		URL    string
		domain string
		hash   uint32
		date   uint64
	)

	for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&id, &title, &URL, &domain, &hash, &date)
		log.Println("article:", id, title, URL, domain, hash, date)

	}
}

func ArticleDisplayContent(db *sql.DB, URL string) {
	row, err := db.Query("SELECT id,title,url,keywords,description,content FROM articles where url=?", URL)
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	//title, url, topimage, keywords, description, content
	var (
		id          int64
		title       string
		url         string
		keywords    string
		description string
		content     string
	)
	for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&id, &title, &url, &keywords, &description, &content)
		fmt.Printf("id:%d\nTitle:%s\nkeywords:%s\ndesc:%s\nURL:%s\n", id, title, keywords, description, url)
		fmt.Printf("----------------------\n%s\n------------------\n", unzip(content))

	}

}

func displayArticlByDay(db *sql.DB, days int) {

	loc, _ := time.LoadLocation("UTC")
	timeend := time.Now().In(loc)
	timestart := beginningofday(timeend)

	for theDay := 0; theDay < days; theDay++ {

		/*
			backHours := time.Hour * time.Duration(-theDay)
			timestart := time.Now().In(loc).Add(backHours).Truncate(time.Hour * 24)
		*/

		row, err := db.Query("SELECT COUNT(*) as count FROM newsarticle WHERE date > ? and date < ? order by date DESC", timestart.Unix(), timeend.Unix())
		ccr := checkCount(row)
		if ccr > 0 {
			fmt.Printf("### %s (%d items)\n", timestart.Format(time.UnixDate), ccr)
		}
		checkErr(err)

		row, err = db.Query("SELECT id,title,URL,domain,hash,date FROM newsarticle WHERE date > ? and date < ? order by date DESC", timestart.Unix(), timeend.Unix())
		if err != nil {
			log.Fatal(err)
		}
		defer row.Close()

		//title, url, domain string, date uint64, hash uint32

		var (
			id     int
			title  string
			URL    string
			domain string
			hash   uint32
			date   uint64
		)

		for row.Next() { // Iterate and fetch the records from result cursor

			row.Scan(&id, &title, &URL, &domain, &hash, &date)
			fmt.Printf("=>%s %s | %s\n", URL, time.Unix(int64(date), 0).UTC().Format("15:04"), title)
		}

		timeend = timestart
		timestart = timeend.AddDate(0, 0, -1)

		if ccr > 0 {
			fmt.Println("")
		}
	}
}

// Utility functions below -------------------------------------------------

func errHandle(err error, debug bool) {
	if err != nil {
		if debug {
			log.Fatal(err)
		} else {
			log.Println(err)
		}
	}
}

func checkCount(rows *sql.Rows) (count int) {
	for rows.Next() {
		err := rows.Scan(&count)
		checkErr(err)
	}
	return count
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func smallhash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func urlhash(URL string) uint32 {
	return (smallhash(fmt.Sprintf(URL)))
}

func beginningofday(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func timetruncate(t time.Time) time.Time {
	return t.Truncate(24 * time.Hour)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func which(bin string) string {
	pathList := []string{"/usr/bin", "/sbin", "/usr/sbin", "/usr/local/bin"}
	for _, p := range pathList {
		if _, err := os.Stat(path.Join(p, bin)); err == nil {
			return path.Join(p, bin)
		}
	}
	return bin
}

func zip(in string) string {
	var b bytes.Buffer
	// gz := gzip.NewWriter(&b)
	gz, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if _, err := gz.Write([]byte(in)); err != nil {
		panic(err)
	}
	if err := gz.Flush(); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	return (b.String())
}

func unzip(in string) string {
	//fmt.Println("ZIP", in)

	b := []byte(in)

	rdata := bytes.NewReader(b)
	r, _ := gzip.NewReader(rdata)
	s, _ := ioutil.ReadAll(r)

	//fmt.Println(string(s))
	return (string(s))
}

func getEpochTime() int64 {
	return time.Now().Unix()
}
