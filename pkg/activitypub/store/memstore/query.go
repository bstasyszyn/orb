/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memstore

import (
	"github.com/trustbloc/orb/pkg/activitypub/store"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type memQuery struct {
	*store.Query
}

func newMemQuery(query *store.Query) *memQuery {
	return &memQuery{
		Query: query,
	}
}

func (q *memQuery) Accept(activity *vocab.ActivityType) bool {
	if len(q.ActivityTypes) > 0 {
		return activity.Type().IsAny(q.ActivityTypes...)
	}

	return true
}

type memQueryResults map[string]*vocab.ActivityType

func (r memQueryResults) Filter(query *store.Query) []*vocab.ActivityType {
	var results []*vocab.ActivityType

	mq := newMemQuery(query)

	for _, a := range r {
		if mq.Accept(a) {
			results = append(results, a)
		}
	}

	return results
}
