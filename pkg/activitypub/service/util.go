/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

func newActivityID(serviceName string) string {
	return fmt.Sprintf("%s/%s", serviceName, uuid.New())
}

func mustMarshal(i interface{}) []byte {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return b
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}

	return u
}
