// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type config struct {
	GithubToken string `json:"github_token"`
	GitlabURL   string `json:"gitlab_url"`
	OrgName     string `json:"org_name"`
	UserAgent   string `json:"user_agent"`
	HookSecret  string `json:"hook_secret"`
	HookListen  string `json:"hook_listen"`
}

func (c *config) parseFile(filepath string) (err error) {
	var f *os.File
	if f, err = os.Open(filepath); err != nil {
		return
	}
	err = json.NewDecoder(f).Decode(&c)
	f.Close()
	return
}

var cfg config

func pushRepo(name, path string) error {
	cmd := exec.Command("git", "push", "--mirror", githubPushURL(name))
	cmd.Dir = gitlabRepoPath(path)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}

func hasCommits(path string) bool {
	cmd := exec.Command("git", "rev-list", "-n", "1", "--all")
	cmd.Dir = gitlabRepoPath(path)
	err := cmd.Start()
	if err != nil {
		return false
	}
	return cmd.Wait() == nil
}

func fullSync() {
	glr, err := getGitlabRepos()
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(glr, func(i, j int) bool { return glr[i].Name < glr[j].Name })

	ghr, err := getGithubRepos()
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(ghr, func(i, j int) bool { return ghr[i].Name < ghr[j].Name })

	i := 0
	for _, repo := range glr {
		// skip repos which have no activity yet or are just shared with the group, not owned by it
		if !strings.HasPrefix(repo.NameWithNamespace, cfg.OrgName+" ") || !hasCommits(repo.Path) {
			fmt.Println("skip:", repo.Name)
			continue
		}
		// skip repos which are on Github but not on Gitlab
		for i < len(ghr) && ghr[i].Name < repo.Name {
			i++
		}

		if i < len(ghr) && ghr[i].Name == repo.Name {
			fmt.Println("exists:", repo.Name)
			err := updateGithubRepo(repo.Name, repo.Description, repo.Path)
			if err != nil {
				log.Println(err)
			}
			i++
		} else {
			fmt.Println("missing:", repo.Name)
			err := createGithubRepo(repo.Name, repo.Description, repo.Path)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func main() {
	err := cfg.parseFile("/etc/github-mirror.json")
	if err != nil {
		log.Fatal(err)
	}

	// make a full sync at startup
	//fullSync()

	// listen for system hooks events
	http.ListenAndServe(cfg.HookListen, new(hooksHandler))
}
