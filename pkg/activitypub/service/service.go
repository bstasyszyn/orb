/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/trustbloc/edge-core/pkg/log"
	"github.com/trustbloc/orb/pkg/activitypub/store"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

var logger = log.New("activitypub_service")

type Behavior interface {
	// This is a decision that the client needs to make synchronously
	HandleFollow(follow *vocab.ActivityType) error
}

type Outbox interface {
	Post(activity *vocab.ActivityType) error
}

type Config struct {
	ListenAddress string
	DomainName    string
	ServiceName   string
}

type Service struct {
	inbox           *InboxHandler
	outbox          *OutboxHandler
	activityHandler *ActivityHandler
}

func NewService(cfg *Config, s store.ActivityStore, behavior Behavior) (*Service, error) {
	undeliverableHandler := NewUndeliverableHandler(cfg.ServiceName)

	outbox, err := NewOutboxHandler(cfg.ServiceName, s, undeliverableHandler)
	if err != nil {
		return nil, err
	}

	activityHandler := NewActivityHandler(cfg.ServiceName, cfg.DomainName+cfg.ServiceName, s, outbox, behavior)

	inbox, err := NewInboxHandler(
		cfg.ServiceName, cfg.ServiceName, cfg.ListenAddress, s, outbox,
		activityHandler,
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		inbox:           inbox,
		outbox:          outbox,
		activityHandler: activityHandler,
	}, nil
}

func (c *Service) Start() {
	c.inbox.Start()
	c.outbox.Start()
}

func (c *Service) Close() {
	c.activityHandler.Close()
	c.outbox.Close()
	c.inbox.Close()
}

func (c *Service) Outbox() Outbox {
	return c.outbox
}

func (h *Service) Subscribe() <-chan *vocab.ActivityType {
	return h.activityHandler.Subscribe()
}

type ActivityHandler struct {
	name        string
	serviceName string
	store       store.ActivityStore
	publisher   Outbox
	behavior    Behavior
	mutex       sync.RWMutex
	subscribers []chan *vocab.ActivityType
}

func NewActivityHandler(name, serviceName string, s store.ActivityStore, publisher Outbox, behavior Behavior) *ActivityHandler {
	return &ActivityHandler{
		name:        name,
		serviceName: serviceName,
		store:       s,
		publisher:   publisher,
		behavior:    behavior,
	}
}

func (h *ActivityHandler) Close() {
	logger.Infof("[%s] Closing activity handler", h.name)

	for _, ch := range h.subscribers {
		close(ch)
	}
}

func (h *ActivityHandler) Subscribe() <-chan *vocab.ActivityType {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	ch := make(chan *vocab.ActivityType, 100) // Make configurable
	h.subscribers = append(h.subscribers, ch)

	return ch
}

func (h *ActivityHandler) notify(activity *vocab.ActivityType) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, ch := range h.subscribers {
		ch <- activity
	}
}

func (h *ActivityHandler) HandleCreateActivity(create *vocab.ActivityType) error {
	logger.Infof("[%s] Handling 'Create' activity: %s", h.name, create.ID())

	h.notify(create)

	return nil
}

func (h *ActivityHandler) HandleFollowActivity(follow *vocab.ActivityType) error {
	logger.Infof("[%s] Handling 'Follow' activity: %s", h.name, follow.ID())

	actor := follow.Actor()
	if actor == nil {
		return fmt.Errorf("no actor specified in 'Follow' activity")
	}

	iri := follow.Object().IRI()
	if iri == nil {
		return fmt.Errorf("no IRI specified in 'object' field of the 'Follow' activity")
	}

	// Make sure that the IRI is targeting this service. If not then ignore the message
	if !strings.Contains(iri.String(), h.serviceName) {
		logger.Infof("[%s] Not handling 'Follow' activity: %s since the object %s is not targeting this service: %s", h.name, follow.ID(), iri, h.serviceName)

		return nil
	}

	if err := h.behavior.HandleFollow(follow); err == nil {
		logger.Infof("[%s] Request for %s to follow %s has been accepted. Replying with 'Accept' activity", h.name, iri, actor)

		if err := h.store.AddFollower(iri, actor); err != nil {
			return errors.WithMessage(err, "unable to store new follower")
		}

		accept := vocab.NewAcceptActivity(h.newActivityID(),
			vocab.NewObjectProperty(vocab.WithActivity(follow)),
			vocab.WithActor(iri),
			vocab.WithTo(actor),
		)

		h.notify(follow)

		logger.Infof("[%s] Publishing 'Accept' activity to %s", h.name, actor)

		if err := h.publisher.Post(accept); err != nil {
			return errors.WithMessagef(err, "unable to reply with 'Accept' to %s", actor)
		}
	} else {
		logger.Infof("[%s] Request for %s to follow %s has been rejected with error: %s. Replying with 'Reject' activity", h.name, iri, actor, err)

		accept := vocab.NewRejectActivity(h.newActivityID(),
			vocab.NewObjectProperty(vocab.WithActivity(follow)),
			vocab.WithActor(iri),
			vocab.WithTo(actor),
		)

		logger.Infof("[%s] Publishing 'Reject' activity to %s", h.name, actor)

		if err := h.publisher.Post(accept); err != nil {
			return errors.WithMessagef(err, "unable to reply with 'Accept' to %s", actor)
		}
	}

	return nil
}

func (h *ActivityHandler) HandleAcceptActivity(accept *vocab.ActivityType) error {
	logger.Infof("[%s] Handling 'Accept' activity: %s", h.name, accept.ID())

	actor := accept.Actor()
	if actor == nil {
		return fmt.Errorf("no actor specified in 'Accept' activity")
	}

	follow := accept.Object().Activity()
	if follow == nil {
		return fmt.Errorf("no 'Follow' activity specified in the 'object' field of the 'Accept' activity")
	}

	if !follow.Type().Is(vocab.TypeFollow) {
		return fmt.Errorf("the 'object' field of the 'Accept' activity must be a 'Follow' type")
	}

	iri := follow.Actor()
	if iri == nil {
		return fmt.Errorf("no actor specified in the original 'Follow' activity of the 'Accept' activity")
	}

	// Make sure that the actor in the original 'Follow' activity is this service. If not then we can ignore the message
	if !strings.Contains(iri.String(), h.serviceName) {
		logger.Infof("[%s] Not handling 'Accept' activity: %s since the actor %s in the original 'Follow' activity is not this service: %s", h.name, accept.ID(), iri, h.serviceName)

		return nil
	}

	if err := h.store.AddFollowing(iri, actor); err != nil {
		return errors.WithMessage(err, "unable to store new following")
	}

	logger.Infof("[%s] %s is now a follower of %s", h.name, iri, actor)

	h.notify(accept)

	return nil
}

func (h *ActivityHandler) HandleRejectActivity(reject *vocab.ActivityType) error {
	logger.Infof("[%s] Handling 'Reject' activity: %s", h.name, reject.ID())

	actor := reject.Actor()
	if actor == nil {
		return fmt.Errorf("no actor specified in 'Reject' activity")
	}

	follow := reject.Object().Activity()
	if follow == nil {
		return fmt.Errorf("no 'Follow' activity specified in the 'object' field of the 'Reject' activity")
	}

	if !follow.Type().Is(vocab.TypeFollow) {
		return fmt.Errorf("the 'object' field of the 'Reject' activity must be a 'Follow' type")
	}

	iri := follow.Actor()
	if iri == nil {
		return fmt.Errorf("no actor specified in the original 'Follow' activity of the 'Reject' activity")
	}

	// Make sure that the actor in the original 'Follow' activity is this service. If not then we can ignore the message
	if !strings.Contains(iri.String(), h.serviceName) {
		logger.Infof("[%s] Not handling 'Reject' activity: %s since the actor %s in the original 'Follow' activity is not this service: %s", h.name, reject.ID(), iri, h.serviceName)

		return nil
	}

	logger.Warnf("[%s] %s rejected our request to follow", h.name, iri)

	h.notify(reject)

	return nil
}

type UndeliverableHandler struct {
	name  string
	mutex sync.Mutex
}

func NewUndeliverableHandler(name string) *UndeliverableHandler {
	return &UndeliverableHandler{
		name: name,
	}
}

func (h *UndeliverableHandler) HandleUndeliverableActivity(id, to string, redeliveryAttempts int, err error, payload []byte) (bool, time.Duration) {
	if redeliveryAttempts == 0 {
		logger.Infof("[%s] Activity was undeliverable. Will attempt to redeliver - ID [%s], To [%s], Redelivery Attempts %d, Payload %s", h.name, id, to, redeliveryAttempts, payload)

		return true, time.Duration((redeliveryAttempts+1)*250) * time.Millisecond
	}

	logger.Infof("[%s] Activity was undeliverable. Will NOT attempt to redeliver - ID [%s], To [%s], Redelivery Attempts %d, Payload %s", h.name, id, to, redeliveryAttempts, payload)

	return false, 0
}

func (h *ActivityHandler) newActivityID() string {
	return newActivityID(h.serviceName)
}
