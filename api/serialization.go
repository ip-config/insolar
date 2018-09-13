/*
 *    Copyright 2018 INS Ecosystem
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

package api

import (
	"github.com/insolar/insolar/core"
	"github.com/pkg/errors"
	"github.com/ugorji/go/codec"
)

func cborMarshal(o interface{}) ([]byte, error) {
	ch := new(codec.CborHandle)
	var data []byte
	err := codec.NewEncoderBytes(&data, ch).Encode(o)
	return data, errors.Wrap(err, "[ CBORMarshal ]")
}

func cborUnMarshal(data []byte, to interface{}) error {
	ch := new(codec.CborHandle)
	err := codec.NewDecoderBytes(data, ch).Decode(&to)
	return errors.Wrap(err, "[ CBORUnMarshal ]")
}

func MarshalArgs(args ...interface{}) (core.Arguments, error) {
	var argsSerialized []byte

	argsSerialized, err := cborMarshal(args)
	if err != nil {
		return nil, errors.Wrap(err, "[ MarshalArgs ]")
	}

	result := core.Arguments(argsSerialized)

	return result, nil
}

func UnMarshalResponse(resp []byte, typeHolders []interface{}) ([]interface{}, error) {
	var marshRes []interface{}
	marshRes = append(marshRes, typeHolders...)

	err := cborUnMarshal(resp, marshRes)
	if err != nil {
		return nil, errors.Wrap(err, "[ UnMarshalResponse ]")
	}

	return marshRes, nil
}
