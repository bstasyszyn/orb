/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ThreeDotsLabs/watermill"
	watermill_http "github.com/ThreeDotsLabs/watermill-http/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/pkg/errors"
	"github.com/trustbloc/orb/pkg/activitypub/service/wmlogger"
	"github.com/trustbloc/orb/pkg/activitypub/store"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type InboxHandler struct {
	name             string
	router           *message.Router
	httpSubscriber   *watermill_http.Subscriber
	msgChannel       <-chan *message.Message
	activityHandlers []interface{}
	activityStore    store.ActivityStore
}

func NewInboxHandler(name, serviceName, listenAddress string, s store.ActivityStore, activityHandlers ...interface{}) (*InboxHandler, error) {
	h := &InboxHandler{
		name:             name,
		activityHandlers: activityHandlers,
		activityStore:    s,
	}

	subscriber := NewGoChannelPubSub(name)

	msgChan, err := subscriber.Subscribe(context.Background(), ActivitiesTopic)
	if err != nil {
		return nil, err
	}

	wmLogger := wmlogger.New()

	httpSubscriber, err := watermill_http.NewSubscriber(
		listenAddress,
		watermill_http.SubscriberConfig{
			UnmarshalMessageFunc: h.unmarshalMessage,
		},
		wmLogger,
	)
	if err != nil {
		return nil, err
	}

	router, err := message.NewRouter(
		message.RouterConfig{},
		wmLogger,
	)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.Recoverer,
		middleware.CorrelationID,
	)

	router.AddPlugin(plugin.SignalsHandler)

	router.AddHandler(
		name,
		serviceName, // this is the URL of our API
		httpSubscriber,
		ActivitiesTopic, // this is the topic the message will be published to
		subscriber,
		func(msg *message.Message) ([]*message.Message, error) {
			logger.Infof("[%s] Got message [%s], fowarding...", name, msg.UUID)

			return message.Messages{msg}, nil
		},
	)

	h.router = router
	h.httpSubscriber = httpSubscriber
	h.msgChannel = msgChan

	return h, nil
}

func (h *InboxHandler) Start() {
	// Start the router
	go h.route()

	// Start the message listener
	go h.listen()

	// HTTP server needs to be started after router is ready.
	<-h.router.Running()

	// Start the HTTP server
	go h.serveHTTP()
}

func (h *InboxHandler) Close() {
	if err := h.httpSubscriber.Close(); err != nil {
		logger.Warnf("Error closing outbox: %s", err)
	}
}

func (h *InboxHandler) Handle(msg *message.Message) {
	logger.Infof("[%s] Handling activities message [%s]: %s", h.name, msg.UUID, msg.Payload)

	activity := &vocab.ActivityType{}

	err := json.Unmarshal(msg.Payload, activity)
	if err != nil {
		logger.Warnf("[%s] Error unmarshalling activity message: %s", h.name, err)

		msg.Nack()

		return
	}

	if err := h.activityStore.PutToInbox(activity); err != nil {
		logger.Errorf("[%s] Error storing activity [%s]: %s", h.name, activity.ID(), err)

		msg.Nack()

		return
	}

	if err := h.handleActivity(activity); err != nil {
		logger.Warnf("[%s] Error handling message: %s", h.name, err)

		msg.Nack()
	} else {
		msg.Ack()
	}
}

func (h *InboxHandler) route() {
	logger.Infof("[%s] Running router on HTTP server...", h.name)

	if err := h.router.Run(context.Background()); err != nil {
		// How to Start error?
		panic(err)
	}

	logger.Infof("[%s] Router is shutting down", h.name)
}

func (h *InboxHandler) listen() {
	for msg := range h.msgChannel {
		logger.Debugf("[%s] Got new message: %s: %s", h.name, msg.UUID, msg.Payload)

		h.Handle(msg)
	}
}

func (h *InboxHandler) serveHTTP() {
	logger.Infof("[%s] Starting HTTP server...", h.name)

	if err := h.httpSubscriber.StartHTTPServer(); err != nil {
		if err == http.ErrServerClosed {
			logger.Infof("[%s] HTTP subscriber has shut down", h.name)
		} else {
			// FIXME: How to handle error?
			panic(err)
		}
	}
}

func (h *InboxHandler) unmarshalMessage(topic string, request *http.Request) (*message.Message, error) {
	b, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read body")
	}

	logger.Debugf("[%s] Unmarshalled message: %s", h.name, b)

	return message.NewMessage(watermill.NewUUID(), b), nil
}

func (h *InboxHandler) handleActivity(activity *vocab.ActivityType) error {
	typeProp := activity.Type()

	switch {
	case typeProp.Is(vocab.TypeCreate):
		return h.handleCreateActivity(activity)
	case typeProp.Is(vocab.TypeFollow):
		return h.handleFollowActivity(activity)
	case typeProp.Is(vocab.TypeAccept):
		return h.handleAcceptActivity(activity)
	case typeProp.Is(vocab.TypeReject):
		return h.handleRejectActivity(activity)
	default:
		return errors.Errorf("unsupported activity type: %s", typeProp.Types())
	}
}

func (h *InboxHandler) handleCreateActivity(activity *vocab.ActivityType) error {
	for _, h := range h.activityHandlers {
		handler, ok := h.(CreateActivityHandler)
		if ok {
			return handler.HandleCreateActivity(activity)
		}
	}

	logger.Warnf("[%s] No handler for 'Create' activity", h.name)

	return nil
}

func (h *InboxHandler) handleFollowActivity(activity *vocab.ActivityType) error {
	for _, h := range h.activityHandlers {
		handler, ok := h.(FollowActivityHandler)
		if ok {
			return handler.HandleFollowActivity(activity)
		}
	}

	logger.Warnf("[%s] No handler for 'Follow' activity", h.name)

	return nil
}

func (h *InboxHandler) handleAcceptActivity(activity *vocab.ActivityType) error {
	for _, h := range h.activityHandlers {
		handler, ok := h.(AcceptActivityHandler)
		if ok {
			return handler.HandleAcceptActivity(activity)
		}
	}

	logger.Warnf("[%s] No handler for 'Accept' activity", h.name)

	return nil
}

func (h *InboxHandler) handleRejectActivity(activity *vocab.ActivityType) error {
	for _, h := range h.activityHandlers {
		handler, ok := h.(RejectActivityHandler)
		if ok {
			return handler.HandleRejectActivity(activity)
		}
	}

	logger.Warnf("[%s] No handler for 'Reject' activity", h.name)

	return nil
}
