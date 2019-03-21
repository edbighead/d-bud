package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/ini.v1"
)

func main() {
	var token = os.Getenv("BITBUCKET_TOKEN")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	var bitbucket = cfg.Section("bitbucket").Key("url").String()
	var project = cfg.Section("bitbucket").Key("project").String()
	var repo = cfg.Section("bitbucket").Key("repo").String()

	var bearer = "Bearer " + token
	var url = bitbucket + "/rest/api/1.0/projects/" + project + "/repos/" + repo + "/pull-requests?state=ALL&withProperties=false&withAttributes=false"

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var pullrequest PR

	json.Unmarshal(body, &pullrequest)

	for _, val := range pullrequest.Values {
		fmt.Println(val)
	}
}
