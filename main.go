package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

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

	//Send an HTTPResponse back to the channel
	ch <- pr
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func AppendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func AppendIfMissingPullRequest(slice []PullRequest, i PullRequest) []PullRequest {
	for _, ele := range slice {
		if reflect.DeepEqual(ele, i) {
			return slice
		}
	}
	return append(slice, i)
}

func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

func (box *Issue) addItem(item PullRequest) []PullRequest {
	box.Pullrequest = append(box.Pullrequest, item)
	return box.Pullrequest
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
	chunkSize := cfg.Section("config").Key("chunks").MustInt(9999)
	jirePrefix := cfg.Section("jira").Key("prefix").String()
	re := regexp.MustCompile(jirePrefix + `-\d*`)
	issues := []string{"IIA-3063", "IIA-3064", "IIA-3080"}

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

	for _, chunk := range divided {
		for _, repo := range chunk {
			url := bitbucket + "/rest/api/1.0/projects/" + project + "/repos/" + repo + "/pull-requests?state=ALL&withProperties=false&withAttributes=false"
			go DoHTTPGet(url, bearer, ch)
		}
	}

	var matchedPRs []PullRequest

	for range repos {
		for _, pr := range (<-ch).Values {
			match := re.FindAllString(pr.Title, -1)
			pr.Issues = match
			if len(match) > 0 {
				for _, m := range match {
					if contains(issues, m) {
						matchedIssues = AppendIfMissing(matchedIssues, m)
						// matchedPRs = append(matchedPRs, pr)
						matchedPRs = AppendIfMissingPullRequest(matchedPRs, pr)
					}
				}
			}
		}
	}

	// noPRs := difference(issues, matchedIssues)
	var fullIssues []Issue
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



	pagesJson, err := json.Marshal(fullIssues)
	if err != nil {
		log.Fatal("Cannot encode to JSON ", err)
	}
	fmt.Fprintf(os.Stdout, "%s", pagesJson)
	elapsed := time.Since(start)
	log.Printf("Took %s", elapsed)
}
