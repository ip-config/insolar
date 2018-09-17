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
	"github.com/insolar/insolar/ledger/index"
	"github.com/insolar/insolar/ledger/record"
)

// Store is used by context unaware clients who can work inside transactions as well as outside.
type Store interface {
	GetRecord(ref *record.Reference) (record.Record, error)
	SetRecord(rec record.Record) (*record.Reference, error)
	GetClassIndex(ref *record.Reference) (*index.ClassLifeline, error)
	SetClassIndex(ref *record.Reference, idx *index.ClassLifeline) error
	GetObjectIndex(ref *record.Reference) (*index.ObjectLifeline, error)
	SetObjectIndex(ref *record.Reference, idx *index.ObjectLifeline) error
}