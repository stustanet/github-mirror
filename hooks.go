// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type systemHook struct {
	Changes []struct {
		Ref string `json:"ref"`
	} `json:"changes"`
	EventName            string `json:"event_name"`
	Name                 string `json:"name"`
	OldPathWithNamespace string `json:"old_path_with_namespace"`
	Path                 string `json:"path"`
	PathWithNamespace    string `json:"path_with_namespace"`
	Project              struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		PathWithNamespace string `json:"path_with_namespace"`
		VisibilityLevel   int    `json:"visibility_level"`
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
		log.Println(id, name, err)
		return
	}
	err = createGithubRepo(details.ID, details.Name, details.Description, details.Path)
	if err != nil {
		log.Println(details.ID, details.Name, err)
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

	switch r.Header.Get("X-Gitlab-Event") {
	case "System Hook":
		var update systemHook
		err := json.NewDecoder(r.Body).Decode(&update)
		if err != nil {
			log.Println(err)
			sendErr(rw, http.StatusBadRequest)
			return
		}

		switch update.EventName {
		case "project_create":
			nsPath := update.PathWithNamespace
			org := nsPath[:len(nsPath)-len(update.Path)-1]
			if org == cfg.OrgName &&
				update.ProjectVisibility == "public" {
				log.Println("create", update.Name)
				mirrorRepo(update.Name, update.ProjectID)
			}

		case "project_destroy":
			nsPath := update.PathWithNamespace
			org := nsPath[:len(nsPath)-len(update.Path)-1]
			if org == cfg.OrgName &&
				update.ProjectVisibility == "public" {
				log.Println("delete", update.Name)
				repos.delete(update.Name)
				if err := deleteGithubRepo(update.ProjectID, update.Name); err != nil {
					log.Println(update.Name, err)
					return
				}
			}

		case "project_transfer":
			if update.ProjectVisibility == "public" {
				nsPath := update.PathWithNamespace
				org := nsPath[:len(nsPath)-len(update.Path)-1]

				oldNsPath := update.OldPathWithNamespace
				oldOrg := oldNsPath[:len(oldNsPath)-len(update.Path)-1]

				if org == cfg.OrgName && org != oldOrg {
					log.Println("create", update.Name)
					mirrorRepo(update.Name, update.ProjectID)
				} else if oldOrg == cfg.OrgName && org != oldOrg {
					log.Println("delete", update.Name)
					repos.delete(update.Name)
					if err := deleteGithubRepo(update.ProjectID, update.Name); err != nil {
						log.Println(update.Name, err)
						return
					}
				}
			}

		case "project_update":
			nsPath := update.PathWithNamespace
			org := nsPath[:len(nsPath)-len(update.Path)-1]
			if org == cfg.OrgName {
				name := update.Name
				mirrored := repos.contains(name)
				if mirrored {
					if update.ProjectVisibility != "public" {
						log.Println("delete", update.ProjectID, name)
						repos.delete(name)
						if err := deleteGithubRepo(update.ProjectID, name); err != nil {
							log.Println(name, err)
							return
						}
					} else {
						log.Println("update", update.ProjectID, name)
						details, err := getGitlabRepo(update.ProjectID)
						if err != nil {
							log.Println(name, err)
							return
						}
						err = updateGithubRepo(update.ProjectID, name, details.Description, details.Path)
						if err != nil {
							log.Println(name, err)
							return
						}
					}
				} else {
					if update.ProjectVisibility == "public" {
						log.Println("create", update.ProjectID, name)
						mirrorRepo(name, update.ProjectID)
					}
				}
			}

		case "repository_update":
			if update.Project.VisibilityLevel == visibilityPublic && update.Project.Namespace == cfg.OrgName {
				name := update.Project.Name
				if len(update.Changes) > 0 {
					log.Println("push", update.ProjectID, name)
					if err := pushRepo(update.ProjectID, name); err != nil {
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
