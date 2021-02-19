/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wmlogger

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/trustbloc/edge-core/pkg/log"
)

const Module = "watermill"

var wmLogger = log.New(Module)

// Logger wraps the TrustBloc logger in a Watermill logger interface
type Logger struct {
	fields watermill.LogFields
}

func New() *Logger {
	return &Logger{}
}

func (l *Logger) Error(msg string, err error, fields watermill.LogFields) {
	wmLogger.Errorf("%s: %s%s", msg, err, l.asString(fields))
}

func (l *Logger) Info(msg string, fields watermill.LogFields) {
	if level := log.GetLevel(Module); level < log.INFO {
		return
	}

	wmLogger.Infof("%s%s", msg, l.asString(fields))
}

func (l *Logger) Debug(msg string, fields watermill.LogFields) {
	if level := log.GetLevel(Module); level < log.DEBUG {
		return
	}

	wmLogger.Debugf("%s%s", msg, l.asString(fields))
}

func (l *Logger) Trace(msg string, fields watermill.LogFields) {
	if level := log.GetLevel(Module); level < log.DEBUG {
		return
	}

	wmLogger.Debugf("%s%s", msg, l.asString(fields))
}

func (l *Logger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &Logger{
		fields: l.fields.Add(fields),
	}
}

func (l *Logger) asString(additionalFields watermill.LogFields) string {
	if len(l.fields) == 0 && len(additionalFields) == 0 {
		return ""
	}

	var msg string

	for k, v := range l.fields.Add(additionalFields) {
		if msg != "" {
			msg += ", "
		}

		var vStr string
		if stringer, ok := v.(fmt.Stringer); ok {
			vStr = stringer.String()
		} else {
			vStr = fmt.Sprintf("%v", v)
		}

		msg += fmt.Sprintf("%s=%s", k, vStr)
	}

	return " - Fields: " + msg
}
