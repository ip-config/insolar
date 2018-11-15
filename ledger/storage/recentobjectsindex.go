/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"sync"

	"github.com/insolar/insolar/core"
)

// RecentObjectsIndex is a base structure
type RecentObjectsIndex struct {
	recentObjects   map[string]*RecentObjectsIndexMeta
	objectLock      sync.Mutex
	pendingRequests map[core.RecordID]struct{}
	requestLock     sync.Mutex
	DefaultTTL      int
}

// RecentObjectsIndexMeta contains meta about indexes
type RecentObjectsIndexMeta struct {
	TTL int
}

// NewRecentObjectsIndex creates default RecentObjectsIndex object
func NewRecentObjectsIndex(defaultTTL int) *RecentObjectsIndex {
	return &RecentObjectsIndex{
		recentObjects:   map[string]*RecentObjectsIndexMeta{},
		pendingRequests: map[core.RecordID]struct{}{},
		DefaultTTL:      defaultTTL,
		objectLock:      sync.Mutex{},
	}
}

// AddID adds object to cache
func (r *RecentObjectsIndex) AddID(id *core.RecordID) {
	r.objectLock.Lock()
	defer r.objectLock.Unlock()

	value, ok := r.recentObjects[string(id.Bytes())]

	if !ok {
		r.recentObjects[string(id.Bytes())] = &RecentObjectsIndexMeta{
			TTL: r.DefaultTTL,
		}
		return
	}

	value.TTL = r.DefaultTTL
}

// AddPendingRequest adds request to cache.
func (r *RecentObjectsIndex) AddPendingRequest(id core.RecordID) {
	r.requestLock.Lock()
	defer r.requestLock.Unlock()

	if _, ok := r.pendingRequests[id]; !ok {
		r.pendingRequests[id] = struct{}{}
		return
	}
}

// GetObjects returns object hot-indexes.
func (r *RecentObjectsIndex) GetObjects() map[string]*RecentObjectsIndexMeta {
	r.objectLock.Lock()
	defer r.objectLock.Unlock()

	targetMap := make(map[string]*RecentObjectsIndexMeta, len(r.recentObjects))
	for key, value := range r.recentObjects {
		targetMap[key] = value
	}

	return targetMap
}

// GetRequests returns request hot-indexes.
func (r *RecentObjectsIndex) GetRequests() []core.RecordID {
	r.requestLock.Lock()
	defer r.requestLock.Unlock()

	requests := make([]core.RecordID, 0, len(r.pendingRequests))
	for id := range r.pendingRequests {
		requests = append(requests, id)
	}

	return requests
}

// ClearZeroTTLObjects clears objects with zero TTL
func (r *RecentObjectsIndex) ClearZeroTTLObjects() {
	r.objectLock.Lock()
	defer r.objectLock.Unlock()

	for key, value := range r.recentObjects {
		if value.TTL == 0 {
			delete(r.recentObjects, key)
		}
	}
}

// ClearObjects clears the whole cache
func (r *RecentObjectsIndex) ClearObjects() {
	r.objectLock.Lock()
	defer r.objectLock.Unlock()

	r.recentObjects = map[string]*RecentObjectsIndexMeta{}
}
