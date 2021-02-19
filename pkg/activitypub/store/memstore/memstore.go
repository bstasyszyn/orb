/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memstore

import (
	"net/url"
	"sync"

	"github.com/trustbloc/edge-core/pkg/log"
	"github.com/trustbloc/orb/pkg/activitypub/store"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

var logger = log.New("activitypub_memstore")

type Store struct {
	name      string
	inbox     map[string]*vocab.ActivityType
	outbox    map[string]*vocab.ActivityType
	following map[string][]*url.URL
	followers map[string][]*url.URL
	mutex     sync.RWMutex
}

func New(name string) *Store {
	return &Store{
		name:      name,
		inbox:     make(map[string]*vocab.ActivityType),
		outbox:    make(map[string]*vocab.ActivityType),
		following: make(map[string][]*url.URL),
		followers: make(map[string][]*url.URL),
	}
}

func (s *Store) PutToInbox(activity *vocab.ActivityType) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Infof("[%s] Storing activity type %s [%s]", s.name, activity.Type(), activity.ID())

	s.inbox[activity.ID()] = activity

	return nil
}

func (s *Store) GetFromInbox(activityID string) (*vocab.ActivityType, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	logger.Infof("[%s] Retrieving activity [%s]", s.name, activityID)

	return s.inbox[activityID], nil
}

func (s *Store) QueryInbox(query *store.Query) (store.QueryResultIterator, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return newQueryResultIterator(memQueryResults(s.inbox).Filter(query)), nil
}

func (s *Store) PutToOutbox(activity *vocab.ActivityType) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Debugf("[%s] Storing activity type %s [%s]", s.name, activity.Type(), activity.ID())

	s.outbox[activity.ID()] = activity

	return nil
}

func (s *Store) GetFromOutbox(activityID string) (*vocab.ActivityType, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	logger.Debugf("[%s] Retrieving activity [%s]", s.name, activityID)

	return s.outbox[activityID], nil
}

func (s *Store) QueryOutbox(query *store.Query) (store.QueryResultIterator, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return newQueryResultIterator(memQueryResults(s.outbox).Filter(query)), nil
}

func (s *Store) AddFollower(actor *url.URL, follower *url.URL) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Debugf("[%s] Adding followers for actor [%s]: %s", s.name, actor, follower)

	actorID := actor.String()

	s.followers[actorID] = append(s.followers[actorID], follower)

	return nil
}

func (s *Store) GetFollowers(actor *url.URL) ([]*url.URL, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	logger.Debugf("[%s] Retrieving followers for actor [%s]", s.name, actor)

	return s.followers[actor.String()], nil
}

func (s *Store) AddFollowing(actor *url.URL, following *url.URL) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Debugf("[%s] Adding following for actor [%s]: %s", s.name, actor, following)

	actorID := actor.String()

	s.following[actorID] = append(s.followers[actorID], following)

	return nil
}

func (s *Store) GetFollowing(actor *url.URL) ([]*url.URL, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	logger.Debugf("[%s] Retrieving following for actor [%s]", s.name, actor)

	return s.following[actor.String()], nil
}
