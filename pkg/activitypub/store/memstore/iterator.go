/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memstore

import (
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type queryResultsIterator struct {
	results []*vocab.ActivityType
	current int
}

func newQueryResultIterator(results []*vocab.ActivityType) *queryResultsIterator {
	return &queryResultsIterator{
		results: results,
		current: -1,
	}
}

func (it *queryResultsIterator) Next() (*vocab.ActivityType, error) {
	if it.current >= len(it.results) {
		return nil, nil
	}

	it.current++

	return it.results[it.current], nil
}

func (it *queryResultsIterator) Close() {
}
