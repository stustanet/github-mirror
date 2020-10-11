// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	visibilityPrivate  = 0
	visibilityInternal = 10
	visibilityPublic   = 20
)

type gitlabRepo struct {
	ID                int   `json:"id"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Description       string `json:"description"`
	Path              string `json:"path"`
}

type gitlabRepos []gitlabRepo

func gitlabRepoPath(id int) string {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(id))
	hash := sha256.Sum256(bs)
	hexDigest := hex.EncodeToString(hash[:])
	return "/var/opt/gitlab/git-data/repositories/@hashed/" + hexDigest[0:2] + "/" + hexDigest[2:4] + "/" + hexDigest + ".git/"
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
}

func getGitlabRepo(id int) (repo gitlabRepo, err error) {
	resp, err := http.Get(cfg.GitlabURL + "api/v4/projects/" + strconv.Itoa(id))
	if err != nil {
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&repo)
	return
}
