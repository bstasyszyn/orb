package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermill_http "github.com/ThreeDotsLabs/watermill-http/pkg/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/pkg/errors"
	"github.com/trustbloc/orb/pkg/activitypub/service/wmlogger"
	"github.com/trustbloc/sidetree-core-go/pkg/canonicalizer"

	"github.com/trustbloc/orb/pkg/activitypub/store"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type OutboxHandler struct {
	name              string
	router            *message.Router
	httpSubscriber    message.Publisher
	publisher         message.Publisher
	handlers          []interface{}
	undeliverableChan <-chan *message.Message
	activityStore     store.ActivityStore
}

func NewOutboxHandler(name string, s store.ActivityStore, handlers ...interface{}) (*OutboxHandler, error) {
	h := &OutboxHandler{
		name:          name,
		handlers:      handlers,
		activityStore: s,
	}

	wmLogger := wmlogger.New()

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		panic(err)
	}

	httpSubscriber, err := watermill_http.NewPublisher(
		watermill_http.PublisherConfig{
			MarshalMessageFunc: h.marshalMessage,
		}, wmLogger)
	if err != nil {
		return nil, err
	}

	publisher := NewGoChannelPubSub("OUTBOX")

	router.AddHandler(
		"h", ActivitiesTopic, publisher, "topic-not-important", httpSubscriber,
		func(msg *message.Message) ([]*message.Message, error) {
			logger.Debugf("[%s] Got message [%s], fowarding...", name, msg.UUID)

			return message.Messages{msg}, nil
		},
	)

	router.AddPlugin(plugin.SignalsHandler)

	undeliverableChan, err := publisher.Subscribe(context.Background(), "undeliverable_queue")
	if err != nil {
		return nil, err
	}

	poisonQueue, err := middleware.PoisonQueue(publisher, "poison_queue")
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		poisonQueue,
	)

	h.router = router
	h.httpSubscriber = httpSubscriber
	h.publisher = publisher
	h.undeliverableChan = undeliverableChan

	return h, nil
}

func (h *OutboxHandler) Start() {
	// Start the redelivery message listener
	go h.handleRedelivery()

	// Start the router
	go h.route()

	// Wait for router to start
	<-h.router.Running()
}

func (h *OutboxHandler) Close() {
	if err := h.httpSubscriber.Close(); err != nil {
		logger.Warnf("[%s] Error closing HTTP subscriber: %s", h.name, err)
	} else {
		logger.Debugf("[%s] Closed HTTP subscriber", h.name)
	}

	if err := h.router.Close(); err != nil {
		logger.Warnf("[%s] Error closing router: %s", h.name, err)
	} else {
		logger.Debugf("[%s] Closed router", h.name)
	}
}

type Response struct {
	err     error
	payload []byte
}

type Request struct {
	respChan chan *Response
	id       string
}

func (h *OutboxHandler) Post(activity *vocab.ActivityType) error {
	if err := h.activityStore.PutToOutbox(activity); err != nil {
		return err
	}

	activityBytes, err := canonicalizer.MarshalCanonical(activity)
	if err != nil {
		return err
	}

	for _, to := range activity.To() {
		msg := message.NewMessage(watermill.NewUUID(), activityBytes)
		msg.Metadata.Set("event_type", ActivitiesTopic)
		msg.Metadata.Set(MetaDataSendTo, to.String())

		logger.Infof("%s Publishing %s", time.Now().String(), ActivitiesTopic)

		// TODO: Don't know if we really need this?
		middleware.SetCorrelationID(activity.ID(), msg)

		if err := h.publisher.Publish(ActivitiesTopic, msg); err != nil {
			// Should we continue with the rest? Or does it post all?
			return err
		}
	}

	return nil
}

func (h *OutboxHandler) route() {
	logger.Infof("Starting router")

	if err := h.router.Run(context.Background()); err != nil {
		panic(err)
	}

	logger.Infof("Router is shutting down")
}

func (h *OutboxHandler) handleRedelivery() {
	for msg := range h.undeliverableChan {
		msg.Ack()

		logger.Warnf("[%s] Got undeliverable message: %+v", h.name, msg)

		// FIXME: How to get error???
		var err error

		h.handleUndeliverableActivity(msg, err)
	}
}

func (h *OutboxHandler) handleUndeliverableActivity(msg *message.Message, err error) {
	var undeliverableHandler UndeliverableActivityHandler

	for _, h := range h.handlers {
		handler, ok := h.(UndeliverableActivityHandler)
		if ok {
			undeliverableHandler = handler
			break
		}
	}

	if undeliverableHandler == nil {
		logger.Warnf("[%s] No handler for undeliverable activity", h.name)

		return
	}

	to := msg.Metadata[MetaDataSendTo]
	activityID := msg.Metadata[middleware.CorrelationIDMetadataKey]

	redeliveryAttempts := 0

	redeliverAttemptsStr, ok := msg.Metadata[MetaDataRedeliveryAttempts]
	if ok {
		ra, err := strconv.Atoi(redeliverAttemptsStr)
		if err != nil {
			logger.Errorf("[%s] Error converting redelivery attempts metadata to number: %s", h.name, redeliverAttemptsStr)
		} else {
			redeliveryAttempts = ra
		}
	}

	if redeliver, delay := undeliverableHandler.HandleUndeliverableActivity(activityID, to, redeliveryAttempts, err, msg.Payload); redeliver {
		logger.Infof("[%s] Redelivering message in %s. Activity ID [%s], To: [%s], Redelivery Attempts: %d", h.name, delay, activityID, to, redeliveryAttempts)

		newMsg := msg.Copy()
		newMsg.Metadata[MetaDataRedeliveryAttempts] = strconv.Itoa(redeliveryAttempts + 1)

		h.publishMessageWithDelay(newMsg, delay)
	} else {
		logger.Infof("[%s] Will not attempt redelivery for message. Activity ID [%s], To: [%s], Redelivery Attempts: %d", h.name, activityID, to, redeliveryAttempts)
	}
}

func (h *OutboxHandler) publishMessageWithDelay(msg *message.Message, delay time.Duration) {
	// FIXME: Shouldn't create a Go routine for each redelivery
	// TODO: Should persist messages in case the server goes down
	go func() {
		time.Sleep(delay)

		logger.Infof("[%s] Attempting to redeliver message [%s]", h.name, msg.UUID)

		if err := h.publisher.Publish(ActivitiesTopic, msg); err != nil {
			logger.Errorf("[%s] Error in redelivery for message [%s]: %s", h.name, msg.UUID, err)
		} else {
			logger.Infof("[%s] Message was redelivered: %+v", h.name, msg)
		}
	}()
}

func (h *OutboxHandler) marshalMessage(url string, msg *message.Message) (*http.Request, error) {
	logger.Debugf("[%s] marshalMessage - URL: %s, Msg: %+v", h.name, url, msg)

	newURL, ok := msg.Metadata[MetaDataSendTo]
	if ok {
		logger.Debugf("[%s] marshalMessage - Sending to new URL: %s", h.name, newURL)

		url = newURL
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(msg.Payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set(watermill_http.HeaderUUID, msg.UUID)

	metadataJson, err := json.Marshal(msg.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal metadata to JSON")
	}

	req.Header.Set(watermill_http.HeaderMetadata, string(metadataJson))

	return req, nil
}
