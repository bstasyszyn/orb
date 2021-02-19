/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"sync"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type MockSubscriber struct {
	serviceName  string
	activityChan <-chan *vocab.ActivityType
	mutex        sync.Mutex
	activities   []*vocab.ActivityType
}

func NewMockSubscriber(serviceName string, activityChan <-chan *vocab.ActivityType) *MockSubscriber {
	return &MockSubscriber{
		serviceName:  serviceName,
		activityChan: activityChan,
	}
}

func (m *MockSubscriber) Listen() {
	for activity := range m.activityChan {
		m.add(activity)
	}
}

func (m *MockSubscriber) add(activity *vocab.ActivityType) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.activities = append(m.activities, activity)
}

func (m *MockSubscriber) Activities() []*vocab.ActivityType {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.activities
}
