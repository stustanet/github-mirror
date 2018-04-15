// Copyright 2018 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package main

import (
	"sync"
)

type hashSet struct {
	set  map[string]struct{}
	lock sync.RWMutex
}

func (hs *hashSet) reset() {
	hs.lock.Lock()
	hs.set = make(map[string]struct{})
	hs.lock.Unlock()
}

func (hs *hashSet) add(item string) {
	hs.lock.Lock()
	hs.set[item] = struct{}{}
	hs.lock.Unlock()
}

func (hs *hashSet) contains(item string) bool {
	hs.lock.RLock()
	_, ok := hs.set[item]
	hs.lock.RUnlock()
	return ok
}

func (hs *hashSet) delete(item string) {
	hs.lock.Lock()
	delete(hs.set, item)
	hs.lock.Unlock()
}
