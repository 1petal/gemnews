package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	goose "github.com/advancedlogic/GoOse"
)

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

func newssGetRaw() {

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
	errHandle(err, false)

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

func newsLoadNewArticles(db *sql.DB, debug bool) {

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
	errHandle(err, debug)

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

		if articleInIndex(db, articlehash, debug) {
			//article already exists in db
			if debug {
				fmt.Println("found", child.Data.URL)
			}
		} else {
			articleInsertIndex(db, child.Data.Title, child.Data.URL, child.Data.Domain, uint64(child.Data.Created), articlehash)
			if debug {
				fmt.Println("Insert into article index", child.Data.URL)
			}
			if articleNoContent(db, child.Data.URL, debug) {
				newsContentRetrieval(db, child.Data.URL, debug)
			}
		}
	}
}

func newsContentRetrieval(db *sql.DB, URL string, debug bool) {

	g := goose.New()
	var retrievalfail bool //default to false
	article, err := g.ExtractFromURL(URL)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Errant URL:", URL)
		return
	}

	if debug {
		println("title:", article.Title)
		println("description:", article.MetaDescription)
		println("keywords:", article.MetaKeywords)
		println("url:", article.FinalURL)
		println("top image:", article.TopImage)
		println("content:", article.CleanedText)
		println("moreContent:", article.AdditionalData)
	}

	if len(article.CleanedText) > 500 {
		articleInsertContent(db, article.Title, URL, article.Domain, article.TopImage, article.MetaKeywords, article.MetaDescription, article.CleanedText, 0)

	} else { ///could not parse out article content. Try fallback method

		articletext := extractor(URL)

		if len(articletext) > 5 {
			articleInsertContent(db, article.Title, URL, article.Domain, article.TopImage, article.MetaKeywords, article.MetaDescription, articletext, 0)
		} else {
			retrievalfail = true
		}
	}

	if retrievalfail { //can't get anything, just store raw content for later processing

		var client http.Client
		resp, err := client.Get(URL)
		if err != nil {
			//log.Fatal(err)
			fmt.Println(err)
			fmt.Println("RAW-Errant URL:", URL)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			// log.Info(bodyString)

			articleInsertContent(db, article.Title, URL, article.Domain, article.TopImage, article.MetaKeywords, article.MetaDescription, bodyString, 1)

		}
	}
}

// search articleindex for URLs that are not in articles table. Attempt to retrieve and store
func newsContentRetry(db *sql.DB, debug bool) {

	//violates seperation!!!

	row, err := db.Query("SELECT URL FROM newsarticle order by date DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()

	//title, url, domain string, date uint64, hash uint32

	var (
		URL  string
		urls []string
	)

	for row.Next() { // Iterate and fetch the records from result cursor

		row.Scan(&URL)
		urls = append(urls, URL)

	}

	for _, item := range urls {

		if articleNoContent(db, item, debug) {
			if len(item) > 5 {
				newsContentRetrieval(db, item, debug)
				fmt.Println("article added:", item)
			}
		} else {
			fmt.Println("article found:", item)
		}
	}
}
