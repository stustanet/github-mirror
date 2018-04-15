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

type repositoryUpdateHook struct {
	Changes []struct {
		Ref string `json:"ref"`
	} `json:"changes"`
	EventName string `json:"event_name"`
	Project   struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		Description       string `json:"description"`
		PathWithNamespace string `json:"path_with_namespace"`
		VisibilityLevel   int    `json:"visibility_level"`
	} `json:"project"`
}

func sendErr(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
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
		var update repositoryUpdateHook
		err := json.NewDecoder(bytes.NewReader(bs)).Decode(&update)
		if err != nil {
			log.Println(err)
			sendErr(rw, http.StatusBadRequest)
			return
		}
		log.Println(update)

		switch update.EventName {
		case "repository_update":
			if update.Project.Namespace == cfg.OrgName {
				name := update.Project.Name
				desc := update.Project.Description

				nsPath := update.Project.PathWithNamespace
				path := nsPath[strings.IndexByte(nsPath, '/')+1:]

				mirrored := repos.contains(name)

				log.Println(name, desc, nsPath, path, mirrored)
				if !mirrored {
					if update.Project.VisibilityLevel == visibilityPublic {
						log.Println("add", name)
						repos.add(name)
						if err := createGithubRepo(name, desc, path); err != nil {
							log.Println(name, err)
						}
					}
				} else {
					if update.Project.VisibilityLevel != visibilityPublic {
						log.Println("delete", name)
						//repos.add(name)
						//createGithubRepo(name, desc, path)
					} else if len(update.Changes) > 0 {
						log.Println("push", name)
						if err := pushRepo(name, path); err != nil {
							log.Println(name, err)
						}
					}
				}
			}
		}
	default:
		sendErr(rw, http.StatusBadRequest)
		return
	}
}
