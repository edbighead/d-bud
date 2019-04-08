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

func GetJiraIDs(jiraURL, jql, token string) ([]string, map[string]IssuePR) {
	var issues []string
	url := jiraURL + "/rest/api/2/search?jql=" + jql
	iprs := make(map[string]IssuePR)

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var jiraResponse JiraResponse

	json.Unmarshal(body, &jiraResponse)

	for _, i := range jiraResponse.Issues {

		key := i.Key

		var oneIssuePR IssuePR
		oneIssuePR.Issue = i

		issues = append(issues, key)
		iprs[key] = oneIssuePR

	}

	return issues, iprs
}

func handleReq(rw http.ResponseWriter, req *http.Request) {
	setupCORS(&rw, req)
	decoder := json.NewDecoder(req.Body)
	var q queryObject
	err := decoder.Decode(&q)
	if err != nil {
		panic(err)
	}
	start := time.Now()

	bitBucketToken := os.Getenv("BITBUCKET_TOKEN")
	jiraToken := os.Getenv("JIRA_TOKEN")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	bitbucket := cfg.Section("bitbucket").Key("url").String()
	project := cfg.Section("bitbucket").Key("project").String()
	repos := strings.Split(cfg.Section("bitbucket").Key("repo").String(), ",")
	chunkSize := cfg.Section("config").Key("chunks").MustInt(9999)
	jiraPrefix := cfg.Section("jira").Key("prefix").String()
	jiraUrl := cfg.Section("jira").Key("url").String()

	re := regexp.MustCompile(jiraPrefix + `-\d*`)
	var bearer = "Bearer " + bitBucketToken
	var authorization = "Basic " + jiraToken
	refs := q.Branch
	jql := q.JQL

	issues, fullIssues := GetJiraIDs(jiraUrl, jql, authorization)

	var matchedIssues []string
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
	var emptyIssue IssuePR

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
						matchedIssues = AppendIfMissing(matchedIssues, m) //array of issues required for release
						matchedPRs = AppendIfMissingPullRequest(matchedPRs, pr)
					}
				}
			}
		}
	}

	for _, matchedIssue := range matchedIssues {
		var i IssuePR
		i.Issue = fullIssues[matchedIssue].Issue
		i.URL = jiraUrl + "/browse/" + fullIssues[matchedIssue].Issue.Key
		for _, pr := range matchedPRs {

			if contains(pr.Issues, matchedIssue) {
				i.addItem(pr)
			}

		}
		fullIssues[matchedIssue] = i
	}

	noPRs := difference(issues, matchedIssues)

	for _, id := range noPRs {
		emptyIssue.Issue = fullIssues[id].Issue
		emptyIssue.URL = jiraUrl + "/browse/" + fullIssues[id].Issue.Key
		fullIssues[id] = emptyIssue
	}

	var transformedIssues []IssuePR
	for _, value := range fullIssues {
		transformedIssues = append(transformedIssues, value)
	}

	pagesJson, err := json.Marshal(transformedIssues)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}

	elapsed := time.Since(start)

	rw.Header().Set("Content-Type", "application/json")

	rw.Write(pagesJson)

	log.Printf("Took %s", elapsed)
}
func setupCORS(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/pullrequests", handleReq).Methods("POST", "OPTIONS")

	log.Fatal(http.ListenAndServe(":8000", router))
}
