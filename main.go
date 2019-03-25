package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
)

func DoHTTPGet(url string, bearer string, ch chan<- Response) {

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var pr Response

	json.Unmarshal(body, &pr)

	ch <- pr
}

func handleReq(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var q queryObject
	err := decoder.Decode(&q)
	if err != nil {
		panic(err)
	}
	// log.Println(q.Branch)

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
	chunkSize := cfg.Section("config").Key("chunks").MustInt(9999)
	jirePrefix := cfg.Section("jira").Key("prefix").String()
	re := regexp.MustCompile(jirePrefix + `-\d*`)

	// temp
	issues := []string{"IIA-3063", "IIA-3064", "IIA-3080"}
	refs := q.Branch

	var matchedIssues []string
	var bearer = "Bearer " + token
	var divided [][]string

	for i := 0; i < len(repos); i += chunkSize {
		end := i + chunkSize

		if end > len(repos) {
			end = len(repos)
		}

		divided = append(divided, repos[i:end])
	}

	var ch chan Response = make(chan Response)

	var matchedPRs []PullRequest
	var fullIssues []Issue
	var emptyIssue Issue

	for _, chunk := range divided {
		for _, repo := range chunk {
			url := bitbucket + "/rest/api/1.0/projects/" + project + "/repos/" + repo + "/pull-requests?state=ALL&withProperties=false&withAttributes=false&at=refs/heads/" + refs
			go DoHTTPGet(url, bearer, ch)
		}
	}
	for range repos {
		for _, pr := range (<-ch).Values {
			match := re.FindAllString(pr.Title, -1)
			pr.Issues = match
			if len(match) > 0 {
				for _, m := range match {
					if contains(issues, m) {
						matchedIssues = AppendIfMissing(matchedIssues, m)
						matchedPRs = AppendIfMissingPullRequest(matchedPRs, pr)
					}
				}
			}
		}
	}

	for _, matchedIssue := range matchedIssues {
		var i Issue
		i.IssueID = matchedIssue
		for _, pr := range matchedPRs {

			if contains(pr.Issues, matchedIssue) {

				i.addItem(pr)
			}
		}
		fullIssues = append(fullIssues, i)
	}

	noPRs := difference(issues, matchedIssues)

	for _, id := range noPRs {
		emptyIssue.IssueID = id
		fullIssues = append(fullIssues, emptyIssue)
	}

	pagesJson, err := json.Marshal(fullIssues)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}

	elapsed := time.Since(start)
	
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(pagesJson)
	
	log.Printf("Took %s", elapsed)
}






func main() {
	router := mux.NewRouter()
	router.HandleFunc("/pullrequests", handleReq).Methods("POST")

	log.Fatal(http.ListenAndServe(":8000", router))
}
