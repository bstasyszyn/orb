/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/edge-core/pkg/log"
	"github.com/trustbloc/orb/pkg/activitypub/service/mocks"
	"github.com/trustbloc/orb/pkg/activitypub/service/wmlogger"
	"github.com/trustbloc/orb/pkg/activitypub/store"
	"github.com/trustbloc/orb/pkg/activitypub/store/memstore"
	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

func TestService(t *testing.T) {
	log.SetLevel(wmlogger.Module, log.WARNING)

	cfg1 := &Config{ServiceName: "/services/service1", DomainName: "http://localhost:8001", ListenAddress: ":8001"}
	store1 := memstore.New(cfg1.ServiceName)
	behavior1 := mocks.NewMockBehavior(cfg1.ServiceName)

	service1, err := NewService(cfg1, store1, behavior1)
	require.NoError(t, err)

	subscriber1 := mocks.NewMockSubscriber(cfg1.ServiceName, service1.Subscribe())
	go subscriber1.Listen()

	cfg2 := &Config{ServiceName: "/services/service2", DomainName: "http://localhost:8002", ListenAddress: ":8002"}
	store2 := memstore.New(cfg2.ServiceName)
	behavior2 := mocks.NewMockBehavior(cfg2.ServiceName)

	service2, err := NewService(cfg2, store2, behavior2)
	require.NoError(t, err)

	subscriber2 := mocks.NewMockSubscriber(cfg2.ServiceName, service2.Subscribe())
	go subscriber2.Listen()

	service1.Start()
	service2.Start()

	service1IRI := mustParseURL("http://localhost:8001/services/service1")
	service2IRI := mustParseURL("http://localhost:8002/services/service2")

	t.Run("Create", func(t *testing.T) {
		const cid = "97bcd005-abb6-423d-a889-18bc1ce84988"

		targetProperty := vocab.NewObjectProperty(vocab.WithObject(
			vocab.NewObject(
				vocab.WithID(cid),
				vocab.WithType(vocab.TypeCAS),
			),
		))

		obj, err := vocab.NewObjectWithDocument(vocab.MustUnmarshalToDoc([]byte(anchorCredential1)))
		if err != nil {
			panic(err)
		}

		unavailableServiceIRI := mustParseURL("http://localhost:8003/services/service3")

		create := vocab.NewCreateActivity(newActivityID(service1IRI.String()),
			vocab.NewObjectProperty(vocab.WithObject(obj)),
			vocab.WithTarget(targetProperty),
			vocab.WithContext(vocab.ContextOrb),
			vocab.WithTo(service1IRI, service2IRI, unavailableServiceIRI),
		)

		require.NoError(t, service1.Outbox().Post(create))

		time.Sleep(500 * time.Millisecond)

		activity, err := store1.GetFromInbox(create.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, create.ID(), activity.ID())

		activity, err = store1.GetFromOutbox(create.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, create.ID(), activity.ID())

		activity, err = store2.GetFromInbox(create.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, create.ID(), activity.ID())

		require.NotEmpty(t, subscriber1.Activities())
		require.NotEmpty(t, subscriber2.Activities())
	})

	t.Run("Follow - Accept", func(t *testing.T) {
		behavior2.AcceptFollowErr = nil

		actorIRI := service1IRI
		targetIRI := service2IRI

		follow := vocab.NewFollowActivity(newActivityID(actorIRI.String()),
			vocab.NewObjectProperty(vocab.WithIRI(targetIRI)),
			vocab.WithActor(actorIRI),
			vocab.WithTo(targetIRI),
		)

		require.NoError(t, service1.Outbox().Post(follow))

		time.Sleep(1000 * time.Millisecond)

		activity, err := store1.GetFromOutbox(follow.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, follow.ID(), activity.ID())

		activity, err = store2.GetFromInbox(follow.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, follow.ID(), activity.ID())

		following, err := store1.GetFollowing(actorIRI)
		require.NoError(t, err)
		require.NotEmpty(t, following)

		ok := false
		for _, f := range following {
			if f.String() == targetIRI.String() {
				ok = true
				break
			}
		}
		require.Truef(t, ok, "expecting %s to be following %s", actorIRI, targetIRI)

		followers, err := store2.GetFollowers(targetIRI)
		require.NoError(t, err)
		require.NotEmpty(t, followers)

		ok = false
		for _, f := range followers {
			if f.String() == actorIRI.String() {
				ok = true
				break
			}
		}
		require.Truef(t, ok, "expecting %s to have %s as a follower", targetIRI, actorIRI)

		// Ensure we have an 'Accept' activity in our inbox
		it, err := store1.QueryInbox(store.NewQuery(store.WithType(vocab.TypeAccept)))
		require.NoError(t, err)
		require.NotNil(t, it)

		ok = false

		for {
			accept, err := it.Next()
			require.NoError(t, err)

			if accept == nil {
				break
			}

			require.True(t, accept.Type().Is(vocab.TypeAccept))

			acceptedFollow := accept.Object().Activity()
			require.NotNil(t, acceptedFollow)

			require.Equal(t, follow.ID(), acceptedFollow.ID())

			acceptBytes, err := json.Marshal(accept)
			require.NoError(t, err)

			t.Logf("Found 'Accept' activity in the inbox: %s", acceptBytes)

			ok = true

			break
		}

		require.Truef(t, ok, "expecting an 'Accept' activity in the inbox")
	})

	t.Run("Follow - Reject", func(t *testing.T) {
		behavior1.AcceptFollowErr = fmt.Errorf("rejecting the follow request")

		actorIRI := service2IRI
		targetIRI := service1IRI

		follow := vocab.NewFollowActivity(newActivityID(actorIRI.String()),
			vocab.NewObjectProperty(vocab.WithIRI(targetIRI)),
			vocab.WithActor(actorIRI),
			vocab.WithTo(targetIRI),
		)

		require.NoError(t, service2.Outbox().Post(follow))

		time.Sleep(1000 * time.Millisecond)

		activity, err := store2.GetFromOutbox(follow.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, follow.ID(), activity.ID())

		activity, err = store1.GetFromInbox(follow.ID())
		require.NoError(t, err)
		require.NotNil(t, activity)
		require.Equal(t, follow.ID(), activity.ID())

		following, err := store2.GetFollowing(actorIRI)
		require.NoError(t, err)

		ok := false
		for _, f := range following {
			if f.String() == targetIRI.String() {
				ok = true
				break
			}
		}
		require.Falsef(t, ok, "expecting %s NOT to be following %s", actorIRI, targetIRI)

		followers, err := store1.GetFollowers(targetIRI)
		require.NoError(t, err)

		ok = false
		for _, f := range followers {
			if f.String() == actorIRI.String() {
				ok = true
				break
			}
		}
		require.Falsef(t, ok, "expecting %s NOT to have %s as a follower", targetIRI, actorIRI)

		// Ensure we have a 'Reject' activity in our inbox
		it, err := store2.QueryInbox(store.NewQuery(store.WithType(vocab.TypeReject)))
		require.NoError(t, err)
		require.NotNil(t, it)

		ok = false

		for {
			reject, err := it.Next()
			require.NoError(t, err)

			if reject == nil {
				break
			}

			require.True(t, reject.Type().Is(vocab.TypeReject))

			rejectedFollow := reject.Object().Activity()
			require.NotNil(t, rejectedFollow)

			require.Equal(t, follow.ID(), rejectedFollow.ID())

			rejectBytes, err := json.Marshal(reject)
			require.NoError(t, err)

			t.Logf("Found 'Reject' activity in the inbox: %s", rejectBytes)

			ok = true

			break
		}

		require.Truef(t, ok, "expecting a 'Reject' activity in the inbox")
	})

	time.Sleep(1000 * time.Millisecond)

	service1.Close()
	service2.Close()

	time.Sleep(1000 * time.Millisecond)
}

const anchorCredential1 = `{
  "@context": [
	"https://www.w3.org/2018/credentials/v1",
	"https://trustbloc.github.io/Context/orb-v1.json"
  ],
  "id": "http://sally.example.com/transactions/bafkreihwsn",
  "type": [
	"VerifiableCredential",
	"AnchorCredential"
  ],
  "issuer": "https://sally.example.com/services/orb",
  "issuanceDate": "2021-01-27T09:30:10Z",
  "credentialSubject": {
	"anchorString": "bafkreihwsn",
	"namespace": "did:orb",
	"version": "1",
	"previousTransactions": {
	  "EiA329wd6Aj36YRmp7NGkeB5ADnVt8ARdMZMPzfXsjwTJA": "bafkreibmrm",
	  "EiABk7KK58BVLHMataxgYZjTNbsHgtD8BtjF0tOWFV29rw": "bafkreibh3w"
	}
  },
  "proofChain": [{}]
}`
