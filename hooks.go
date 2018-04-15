// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type systemHook struct {
	Changes []struct {
		Ref string `json:"ref"`
	} `json:"changes"`
	EventName string `json:"event_name"`
	Name      string `json:"name"`
	OwnerName string `json:"owner_name"`
	Path      string `json:"path"`
	Project   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		//Description       string `json:"description"`
		PathWithNamespace string `json:"path_with_namespace"`
		//VisibilityLevel   int    `json:"visibility_level"`
	} `json:"project"`
	ProjectID         int    `json:"project_id"`
	ProjectVisibility string `json:"project_visibility"`
}

func sendErr(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func mirrorRepo(name string, id int) {
	details, err := getGitlabRepo(id)
	if err != nil {
		log.Println(name, err)
		return
	}
	err = createGithubRepo(details.Name, details.Description, details.Path)
	if err != nil {
		log.Println(details.Name, err)
		return
	}
	repos.add(details.Name)
}

type hooksHandler struct{}

func (h *hooksHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/mirror" {
		sendErr(rw, http.StatusTeapot)
		return
	}

	if r.Header.Get("X-Gitlab-Token") != cfg.HookSecret {
		sendErr(rw, http.StatusForbidden)
		return
	}

	log.Println(r.Header)
	bs, err := ioutil.ReadAll(r.Body)
	log.Println(string(bs), err)

	switch r.Header.Get("X-Gitlab-Event") {
	case "System Hook":
		var update systemHook
		err := json.NewDecoder(bytes.NewReader(bs)).Decode(&update)
		if err != nil {
			log.Println(err)
			sendErr(rw, http.StatusBadRequest)
			return
		}
		log.Println(update)

		switch update.EventName {
		case "project_create":
			if update.OwnerName == cfg.OrgName &&
				update.ProjectVisibility == "public" {
				log.Println("create", update.Name)
				mirrorRepo(update.Name, update.ProjectID)
			}

		case "project_destroy":
			if update.OwnerName == cfg.OrgName &&
				update.ProjectVisibility == "public" {
				log.Println("delete", update.Name)
				repos.delete(update.Name)
				if err := deleteGithubRepo(update.Name); err != nil {
					log.Println(update.Name, err)
					return
				}
			}

		case "project_update":
			if update.OwnerName == cfg.OrgName {
				name := update.Name
				mirrored := repos.contains(name)
				if mirrored {
					if update.ProjectVisibility != "public" {
						log.Println("delete", name)
						repos.delete(name)
						if err := deleteGithubRepo(name); err != nil {
							log.Println(name, err)
							return
						}
					} else {
						details, err := getGitlabRepo(update.ProjectID)
						if err != nil {
							log.Println(name, err)
							return
						}
						err = updateGithubRepo(name, details.Description, details.Path)
						if err != nil {
							log.Println(name, err)
							return
						}
					}
				} else {
					if update.ProjectVisibility == "public" {
						log.Println("create", name)
						mirrorRepo(name, update.ProjectID)
					}
				}
			}

		case "repository_update":
			if update.Project.Namespace == cfg.OrgName {
				name := update.Project.Name
				nsPath := update.Project.PathWithNamespace
				path := nsPath[strings.IndexByte(nsPath, '/')+1:]
				if len(update.Changes) > 0 {
					log.Println("push", name)
					if err := pushRepo(name, path); err != nil {
						log.Println(name, err)
						return
					}
				}
			}
		}
	default:
		sendErr(rw, http.StatusBadRequest)
		return
	}
}
