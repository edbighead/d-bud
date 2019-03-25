package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

func DoHTTPGet(url string, bearer string, ch chan<- PullRequest) {

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var pr PullRequest

	json.Unmarshal(body, &pr)

	//Send an HTTPResponse back to the channel
	ch <- pr
}

func main() {
	start := time.Now()

	token := os.Getenv("BITBUCKET_TOKEN")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	bitbucket := cfg.Section("bitbucket").Key("url").String()
	project := cfg.Section("bitbucket").Key("project").String()
	repos := strings.Split(cfg.Section("bitbucket").Key("repo").String(), ",")

	var divided [][]string

	chunkSize := 5

	for i := 0; i < len(repos); i += chunkSize {
		end := i + chunkSize

		if end > len(repos) {
			end = len(repos)
		}

		divided = append(divided, repos[i:end])
	}

	var ch chan PullRequest = make(chan PullRequest)

	for _, chunk := range divided {
		for _, repo := range chunk {
			bearer := "Bearer " + token
			url := bitbucket + "/rest/api/1.0/projects/" + project + "/repos/" + repo + "/pull-requests?state=ALL&withProperties=false&withAttributes=false"
			go DoHTTPGet(url, bearer, ch)
		}
	}

	for range repos {
		// Use the response (<-ch).body
		fmt.Println((<-ch).Values)
	}

	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
}
