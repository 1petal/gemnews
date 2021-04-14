package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func getnews() {

	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://reddit.com/r/news.json", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "test article reader")
	rsp, err := client.Do(req)

	if rsp.StatusCode != http.StatusOK {
		log.Fatal(rsp.Status)
	}

	//	_, err = io.Copy(os.Stdout, rsp.Body)
	if err != nil {
		log.Fatal(err)
	}

	resp := new(Response)
	err = json.NewDecoder(rsp.Body).Decode(resp)

	for _, child := range resp.Data.Children {
		fmt.Printf("###%s\n  %s\n", child.Data.Title, time.Unix(int64(child.Data.Created), 0).UTC().Format(time.UnixDate))
		//fmt.Println("[", child.Data.Created, child.Data.Title)
		fmt.Println("=>", child.Data.URL, child.Data.Domain)
		fmt.Println("")
	}

	/*
		fmt.Println()
		fmt.Println("###Links")
		fmt.Println()

		for _, child := range resp.Data.Children {
			fmt.Println("=>", child.Data.URL, child.Data.Title)
		}
	*/
}

func loadNewArticles(db *sql.DB, debug bool) {

	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://reddit.com/r/news.json", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "test article reader")
	rsp, err := client.Do(req)

	if rsp.StatusCode != http.StatusOK {
		log.Fatal(rsp.Status)
	}

	//	_, err = io.Copy(os.Stdout, rsp.Body)
	if err != nil {
		log.Fatal(err)
	}

	resp := new(Response)
	err = json.NewDecoder(rsp.Body).Decode(resp)

	/*
		for _, child := range resp.Data.Children {
			fmt.Printf("###%s\n  %s\n", child.Data.Title, time.Unix(int64(child.Data.Created), 0).UTC().Format(time.UnixDate))
			//fmt.Println("[", child.Data.Created, child.Data.Title)
			fmt.Println("=>", child.Data.URL, child.Data.Domain)
			fmt.Println("")
		}
	*/

	for _, child := range resp.Data.Children {

		articlehash := smallhash(fmt.Sprintf("%s%f", child.Data.Title, child.Data.Created))

		if artexist(db, articlehash, debug) {
			//article already exists in db
			if debug {
				fmt.Println("found", child.Data.URL)
			}
		} else {
			insertarticle(db, child.Data.Title, child.Data.URL, child.Data.Domain, uint64(child.Data.Created), articlehash)
			if debug {
				fmt.Println("INSERT", child.Data.URL)
			}
		}
	}
}

type Item struct {
	Title   string
	URL     string
	Created float64 `orm:"column(created_utc);type(datetime)" json:"created_utc"`
	Domain  string
}

type Response struct {
	Data struct {
		Children []struct {
			Data Item
		}
	}
}
