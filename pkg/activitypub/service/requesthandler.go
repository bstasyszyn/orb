/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"sync"
)

type requestHandler struct {
	requests map[string]*Response
	mutex    sync.RWMutex
}
