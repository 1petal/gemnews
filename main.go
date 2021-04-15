package main

import (
	"flag"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
	//_ "modernc.org/sqlite"
)

func main() {

	var dbpath string
	debug := false

	// Command line flags control behavior
	debugPtr := flag.Bool("debug", false, "Debug flag")
	flag.StringVar(&dbpath, "db", "sqlite.db", "SQLite database path")
	loadptr := flag.Bool("l", false, "Load articles flag")
	dumpptr := flag.Bool("dump", false, "dump database contents")
	cretryptr := flag.Bool("cretry", false, "Content Databse load/retry")
	daysPtr := flag.Int("days", 5, "Days to display")

	flag.Parse()

	if *debugPtr {
		fmt.Println("DEBUG mode")
		debug = true
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

	} else {

		displayArticlByDay(db, *daysPtr)
	}
}
