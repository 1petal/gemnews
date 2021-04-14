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
	debugPtr := flag.Bool("d", false, "Debug flag")
	flag.StringVar(&dbpath, "db", "sqlite.db", "SQLite database path")
	loadptr := flag.Bool("l", false, "Load articles flag")
	dumpptr := flag.Bool("dump", false, "dump database contents")

	flag.Parse()

	if *debugPtr {
		fmt.Println("DEBUG mode")
		debug = true
	}

	db := dbinit(dbpath, debug)
	defer db.Close()

	if *loadptr {
		// Load them articles!

		loadNewArticles(db, debug)

	} else if *dumpptr {
		displayArticleAll(db)

	} else {

		displayArticlByDay(db)
	}
}
