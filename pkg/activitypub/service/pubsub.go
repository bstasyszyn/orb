/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type GoChannelPubSub struct {
	name     string
	msgChans map[Topic]chan *message.Message
	mutex    sync.RWMutex
	ackChan  chan []*message.Message
}

func NewGoChannelPubSub(name string) *GoChannelPubSub {
	m := &GoChannelPubSub{
		name:     name,
		msgChans: make(map[Topic]chan *message.Message),
		ackChan:  make(chan []*message.Message, 100), // TODO: Make configurable
	}

	go m.processAcksNacks()

	return m
}

func (p *GoChannelPubSub) Publish(topic string, messages ...*message.Message) error {
	p.mutex.RLock()
	msgChan := p.msgChans[Topic(topic)]
	p.mutex.RUnlock()

	for _, msg := range messages {
		logger.Debugf("[%s] GoChannelPubSub.Post - %+v", p.name, msg)

		msgChan <- msg
	}

	p.ackChan <- messages

	return nil
}

func (p *GoChannelPubSub) processAcksNacks() {
	for messages := range p.ackChan {
		p.checkAckNack(messages)
	}
}

func (p *GoChannelPubSub) checkAckNack(messages []*message.Message) {
	// TODO: Make configurable
	timeout := 2 * time.Second

	for _, msg := range messages {
		logger.Debugf("[%s] Checking for Ack/Nack on message: %+v", p.name, msg)

		select {
		case <-msg.Acked():
			logger.Debugf("[%s] Message was sent %+v", p.name, msg)

		case <-msg.Nacked():
			logger.Infof("[%s] Message could not be sent %+v", p.name, msg)

			p.mutex.RLock()
			undeliverableChan, ok := p.msgChans["undeliverable_queue"]
			p.mutex.RUnlock()

			if ok {
				logger.Infof("[%s] Adding message to undeliverable queue %+v", p.name, msg)

				undeliverableChan <- msg
			}

		case <-time.After(timeout):
			logger.Warnf("[%s] Timed out after %s waiting for Ack/Nack on message: %+v", p.name, timeout, msg)
		}
	}
}

func (p *GoChannelPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	logger.Debugf("[%s] GoChannelPubSub.Subscribe to topic [%s]", p.name, topic)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	msgChan, ok := p.msgChans[Topic(topic)]
	if !ok {
		msgChan = make(chan *message.Message, 100)
		p.msgChans[Topic(topic)] = msgChan
	}

	return msgChan, nil
}

func (p *GoChannelPubSub) Close() error {
	for _, msgChan := range p.msgChans {
		close(msgChan)
	}

	close(p.ackChan)

	return nil
}
