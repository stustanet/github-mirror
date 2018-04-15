// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	//"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type pushHook struct {
	Project struct {
		Name      string
		Namespace string
	}
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

	switch r.Header.Get("X-Gitlab-Event") {
	case "Repository Update Hook":
		// WIP: dump request
		log.Println(r.Header)
		bs, err := ioutil.ReadAll(r.Body)
		log.Println(string(bs), err)
	case "Push Hook":
		// WIP: dump request
		log.Println(r.Header)
		bs, err := ioutil.ReadAll(r.Body)
		log.Println(string(bs), err)
	default:
		sendErr(rw, http.StatusBadRequest)
		return
	}
}
