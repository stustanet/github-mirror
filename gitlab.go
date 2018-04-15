// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	visibilityPrivate  = 0
	visibilityInternal = 10
	visibilityPublic   = 20
)

type gitlabRepos []struct {
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Description       string `json:"description"`
	Path              string `json:"path"`
}

func gitlabRepoPath(path string) string {
	return "/var/opt/gitlab/git-data/repositories/" + cfg.OrgName + "/" + path + ".git/"
}

func getGitlabRepos() (repos gitlabRepos, err error) {
	page := 1
	for {
		var newRepos gitlabRepos
		resp, err := http.Get(cfg.GitlabURL + "api/v4/groups/" + cfg.OrgName +
			"/projects?visibility=public&simple=1&per_page=100&page=" + strconv.Itoa(page))
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
	return
}
