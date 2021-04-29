package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func extractor(sourceurl string) string {

	var (
		articletext  []string
		articlelinks []string
		debug        bool
	)

	response, err := http.Get(sourceurl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	textTags := []string{
		"a",
		"p", "span", "em", "string", "blockquote", "q", "cite",
		"h1", "h2", "h3", "h4", "h5", "h6",
	}

	tag := ""
	attrs := map[string]string{}
	enter := false

	tokenizer := html.NewTokenizer(response.Body)
	for {
		tt := tokenizer.Next()
		token := tokenizer.Token()

		err := tokenizer.Err()
		if err == io.EOF {
			break
		}

		switch tt {
		case html.ErrorToken:
			log.Fatal(err)
		case html.StartTagToken, html.SelfClosingTagToken:
			enter = false
			attrs = map[string]string{}

			tag = token.Data
			for _, ttt := range textTags {
				if tag == ttt {
					enter = true
					for _, attr := range token.Attr {
						attrs[attr.Key] = attr.Val
					}
					break
				}
			}
		case html.TextToken:
			if enter {
				data := strings.TrimSpace(token.Data)

				if len(data) > 0 {
					switch tag {
					case "a":
						// fmt.Printf("[%s](%s)\n", data, attrs["href"])
						if len(strings.Fields(data)) < 4 {
							articletext = append(articletext, fmt.Sprintf("[%s] ", data))
						} else {
							articletext = append(articletext, fmt.Sprintf("%s ", data))
						}
						// fmt.Printf("=> %s %s\n", attrs["href"], data)
						articlelinks = append(articlelinks, fmt.Sprintf("=> %s %s", attrs["href"], data))
						//fmt.Printf("[%s]", data)
					case "h1":
						if debug {
							fmt.Printf("\n\n# %s\n\n", data)
						}
						articletext = append(articletext, fmt.Sprintf("\n\n## %s\n\n", data))
					case "h2", "h3":
						if debug {
							fmt.Printf("\n## %s\n", data)
						}
						articletext = append(articletext, fmt.Sprintf("\n\n## %s\n", data))
					case "h4", "h5", "h6":
						if debug {
							fmt.Printf("\n### %s\n", data)
						}
						articletext = append(articletext, fmt.Sprintf("\n### %s\n", data))
					default:
						if debug {
							fmt.Println(data)
						}
						articletext = append(articletext, fmt.Sprintf("%s\n", data))
						//fmt.Println("DEBUG: li", strings.LastIndex(data, "."), "len", len(data))
						if strings.LastIndex(data, ".") == len(data)-1 {
							articletext = append(articletext, fmt.Sprintf("\n"))
						}

					}
				}
			}
		}
	}
	//append article links to end of article text

	articletext = append(articletext, fmt.Sprintf("\n\n---\n"))
	for _, links := range articlelinks {
		if strings.HasPrefix(links, "=> https:") || strings.HasPrefix(links, "=> http:") || strings.HasPrefix(links, "=> gemini:") || strings.HasPrefix(links, "=> gopher:") {
			articletext = append(articletext, fmt.Sprintf("%s\n", links))
		}
	}
	return strings.Join(articletext, "")
}
