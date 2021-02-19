/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"sync"
	"time"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

//func TestHandler(t *testing.T) {
//	store1 := store.NewMemStore("INBOX-1")
//	activityHandler1 := newMockActivityHandler("INBOX-1")
//	//activityHandler1 := newMockActivityHandler("INBOX-1").
//	//	WithError(fmt.Errorf("injected handler error for INBOX-1"))
//	inbox1, err := NewInboxHandler("INBOX-1", "/services/service1", ":8001", store1, activityHandler1)
//	require.NoError(t, err)
//
//	store2 := store.NewMemStore("INBOX-2")
//	activityHandler2 := newMockActivityHandler("INBOX-2")
//	inbox2, err := NewInboxHandler("INBOX-2", "/services/service2", ":8002", store2, activityHandler2)
//	require.NoError(t, err)
//
//	inbox1.Start()
//
//	undeliverableActivityHandler := newMockUndeliverableHandler("OUTBOX")
//
//	outboxStore := store.NewMemStore("OUTBOX")
//	outbox, err := NewOutboxHandler("OUTBOX", outboxStore, undeliverableActivityHandler)
//	require.NoError(t, err)
//
//	outbox.Start()
//
//	if err := outbox.Post(newCreateActivity()); err != nil {
//		panic(err)
//	}
//
//	t.Run("Follow", func(t *testing.T) {
//		actorIRI := mustParseURL("http://localhost:8001/services/service1")
//		objectIRI := mustParseURL("http://localhost:8002/services/service2")
//
//		follow := vocab.NewFollowActivity(newActivityID(actorIRI.String()),
//			vocab.NewObjectProperty(vocab.WithIRI(objectIRI)),
//			vocab.WithActor(actorIRI),
//			vocab.WithTo(objectIRI),
//		)
//
//		if err := outbox.Post(follow); err != nil {
//			panic(err)
//		}
//	})
//
//	time.Sleep(500 * time.Millisecond)
//
//	inbox2.Start()
//
//	time.Sleep(2000 * time.Millisecond)
//
//	outbox.Close()
//
//	inbox1.Close()
//	inbox2.Close()
//
//	time.Sleep(2 * time.Second)
//
//	require.Len(t, activityHandler1.Received(), 2)
//	require.Len(t, activityHandler2.Received(), 1)
//	//require.Len(t, undeliverableActivityHandler.Received(), 2)
//}

type mockActivityHandler struct {
	name     string
	err      error
	mutex    sync.Mutex
	received []*vocab.ActivityType
}

func newMockActivityHandler(name string) *mockActivityHandler {
	return &mockActivityHandler{name: name}
}

func (h *mockActivityHandler) WithError(err error) *mockActivityHandler {
	h.err = err

	return h
}

func (h *mockActivityHandler) HandleCreateActivity(activity *vocab.ActivityType) error {
	logger.Infof("[%s] Handling create activity: %s", h.name, activity.ID())

	h.addActivity(activity)

	return h.err
}

func (h *mockActivityHandler) HandleFollowActivity(activity *vocab.ActivityType) error {
	logger.Infof("[%s] Handling follow activity: %s", h.name, activity.ID())

	h.addActivity(activity)

	return h.err
}

func (h *mockActivityHandler) addActivity(activity *vocab.ActivityType) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.received = append(h.received, activity)
}

func (h *mockActivityHandler) Received() []*vocab.ActivityType {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.received
}

type info struct {
	id      string
	to      string
	payload []byte
}

type mockUndeliverableHandler struct {
	name     string
	mutex    sync.Mutex
	received []*info
}

func newMockUndeliverableHandler(name string) *mockUndeliverableHandler {
	return &mockUndeliverableHandler{name: name}
}

func (h *mockUndeliverableHandler) HandleUndeliverableActivity(id, to string, redeliveryAttempts int, err error, payload []byte) (bool, time.Duration) {
	h.addActivity(id, to, payload)

	if redeliveryAttempts <= 2 {
		logger.Infof("[%s] Activity was undeliverable. Will attempt to redeliver - ID [%s], To [%s], Redelivery Attempts %d, Payload %s", h.name, id, to, redeliveryAttempts, payload)

		return true, time.Duration((redeliveryAttempts+1)*250) * time.Millisecond
	}

	logger.Infof("[%s] Activity was undeliverable. Will NOT attempt to redeliver - ID [%s], To [%s], Redelivery Attempts %d, Payload %s", h.name, id, to, redeliveryAttempts, payload)

	return false, 0
}

func (h *mockUndeliverableHandler) addActivity(activityID, to string, payload []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.received = append(h.received, &info{
		id:      activityID,
		to:      to,
		payload: payload,
	})
}

func (h *mockUndeliverableHandler) Received() []*info {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.received
}
