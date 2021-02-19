/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"time"

	"github.com/trustbloc/orb/pkg/activitypub/vocab"
)

type Topic = string

const (
	ActivitiesTopic Topic = "activities"

	MetaDataSendTo             = "send_to"
	MetaDataRedeliveryAttempts = "redelivery_attempts"
)

type CreateActivityHandler interface {
	HandleCreateActivity(activity *vocab.ActivityType) error
}

type FollowActivityHandler interface {
	HandleFollowActivity(activity *vocab.ActivityType) error
}

type AcceptActivityHandler interface {
	HandleAcceptActivity(activity *vocab.ActivityType) error
}

type RejectActivityHandler interface {
	HandleRejectActivity(activity *vocab.ActivityType) error
}

type UndeliverableActivityHandler interface {
	HandleUndeliverableActivity(id, to string, redeliveryAttempts int, err error, payload []byte) (bool, time.Duration)
}
