/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"net/url"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type ActivityStore interface {
	PutToInbox(activity *vocab.ActivityType) error
	GetFromInbox(activityID string) (*vocab.ActivityType, error)
	QueryInbox(query *Query) (QueryResultIterator, error)
	PutToOutbox(activity *vocab.ActivityType) error
	GetFromOutbox(activityID string) (*vocab.ActivityType, error)
	QueryOutbox(query *Query) (QueryResultIterator, error)
	AddFollower(actor *url.URL, follower *url.URL) error
	GetFollowers(actor *url.URL) ([]*url.URL, error)
	AddFollowing(actor *url.URL, following *url.URL) error
	GetFollowing(actor *url.URL) ([]*url.URL, error)
}

type Query struct {
	ActivityTypes []vocab.Type
}

type QueryOpt func(q *Query)

func NewQuery(opts ...QueryOpt) *Query {
	q := &Query{}

	for _, opt := range opts {
		opt(q)
	}

	return q
}

func WithType(t vocab.Type) QueryOpt {
	return func(query *Query) {
		query.ActivityTypes = append(query.ActivityTypes, t)
	}
}

type QueryResultIterator interface {
	Next() (*vocab.ActivityType, error)
	Close()
}
