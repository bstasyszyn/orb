/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type MockBehavior struct {
	serviceName     string
	AcceptFollowErr error
}

func NewMockBehavior(serviceName string) *MockBehavior {
	return &MockBehavior{serviceName: serviceName}
}

func (m *MockBehavior) HandleFollow(follow *vocab.ActivityType) error {
	return m.AcceptFollowErr
}
