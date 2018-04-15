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
)

type repositoryUpdateHook struct {
	Changes []struct {
		Ref string `json:"ref"`
	} `json:"changes"`
	EventName         string `json:"event_name"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	ProjectVisibility string `json:"project_visibility"`
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
	case "Repository Update Hook":
		var update repositoryUpdateHook
		err := json.NewDecoder(bytes.NewReader(bs)).Decode(&update)
		if err != nil {
			log.Println(err)
			sendErr(rw, http.StatusBadRequest)
			return
		}

		log.Println(update)
	default:
		sendErr(rw, http.StatusBadRequest)
		return
	}
}
