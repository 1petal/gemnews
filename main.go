package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	//_ "modernc.org/sqlite"
)

func main() {

	var dbpath, extracturl string
	debug := false

	cgiurl := os.Getenv("URLREQUESTED")

	// Command line flags control behavior
	debugPtr := flag.Bool("debug", false, "Debug flag")
	flag.StringVar(&dbpath, "db", "sqlite.db", "SQLite database path")
	loadptr := flag.Bool("l", false, "Load articles flag")
	dumpptr := flag.Bool("dump", false, "dump database contents")
	cretryptr := flag.Bool("cretry", false, "Content Databse load/retry")
	daysPtr := flag.Int("days", 5, "Days to display")
	flag.StringVar(&extracturl, "extract", "", "extract markdown from url")

	flag.Parse()

	if *debugPtr {
		fmt.Println("DEBUG mode")
		debug = true

		// fetcha all env variables
		fmt.Printf("\n\n--ENVIRONMENT--")
		for _, element := range os.Environ() {
			variable := strings.Split(element, "=")
			fmt.Println(variable[0], "-->", variable[1])
		}
		fmt.Println()
	}

	db := dbinit(dbpath, debug)
	defer db.Close()

	if *loadptr {
		// Load them articles!

		newsLoadNewArticles(db, debug)

	} else if *dumpptr {
		displayArticleAll(db)

	} else if *cretryptr {
		newsTablesInit(db, debug)
		newsContentRetry(db, debug)

	} else if len(extracturl) > 5 {
		fmt.Println("call to extract", extracturl)
		fmt.Println(extractor(extracturl))

	} else if len(cgiurl) > 5 {

		// cgiurl := os.Getenv("URLREQUESTED")

		u, err := url.Parse(cgiurl)
		if err != nil {
			panic(err)
		}
		m, _ := url.ParseQuery(u.RawQuery)

		if debug {
			fmt.Println("cgiurl", cgiurl)
			fmt.Println("parsed paramaters:", m)
		}

		if len(m["a"]) > 0 {
			rawhash := strings.Join(m["a"], "")

			ArticleDisplayContentByHash(db, rawhash)
		} else {
			displayArticlByDay(db, *daysPtr)
		}

	} else {

		displayArticlByDay(db, *daysPtr)
	}
}
