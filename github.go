// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type githubRepos []struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type createRepo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	HasIssues   bool   `json:"has_issues"`
	HasProjects bool   `json:"has_projects"`
	HasWiki     bool   `json:"has_wiki"`
}

type updateRepo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func setGithubHeaders(r *http.Request) {
	r.Header.Set("Accept", "application/vnd.github.v3+json")
	r.Header.Set("Authorization", "token "+cfg.GithubToken)
	r.Header.Set("User-Agent", cfg.UserAgent)
}

func githubPushURL(name string) string {
	return "https://" + cfg.GithubToken + "@github.com/" + cfg.OrgName + "/" + name + ".git"
}

func getGithub(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/"+path, nil)
	if err != nil {
		return nil, err
	}
	setGithubHeaders(req)
	return http.DefaultClient.Do(req)
}

func sendGithub(method, path string, jsonData interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(jsonData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, "https://api.github.com/"+path, &buf)
	if err != nil {
		return nil, err
	}
	setGithubHeaders(req)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	return http.DefaultClient.Do(req)
}

func getGithubRepos() (repos githubRepos, err error) {
	page := 1
	for {
		var newRepos githubRepos
		resp, err := getGithub("orgs/" + cfg.OrgName +
			"/repos?type=public&per_page=100&page=" + strconv.Itoa(page))
		if err != nil {
			return nil, err
		}
		if err = json.NewDecoder(resp.Body).Decode(&newRepos); err != nil {
			return nil, err
		}
		if len(newRepos) < 100 {
			if page == 1 {
				return newRepos, nil
			} else {
				return append(repos, newRepos...), nil
			}
		} else {
			repos = append(repos, newRepos...)
			page++
		}
	}
}

func createGithubRepo(name, description, path string) error {
	cr := createRepo{
		Name:        name,
		Description: description,
		Homepage:    cfg.GitlabURL + cfg.OrgName + "/" + name,
		HasIssues:   true,
		HasProjects: false,
		HasWiki:     false,
	}
	resp, err := sendGithub("POST", "orgs/"+cfg.OrgName+"/repos", cr)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		bs, err := ioutil.ReadAll(resp.Body)
		log.Println(string(bs), err)
		return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, resp.Status)
	}
	resp.Body.Close()
	return pushRepo(name, path)
}

func updateGithubRepo(name, description, path string) error {
	ur := updateRepo{
		Name:        name,
		Description: description,
	}
	resp, err := sendGithub("PATCH", "repos/"+cfg.OrgName+"/"+name, ur)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}
	return pushRepo(name, path)
}
