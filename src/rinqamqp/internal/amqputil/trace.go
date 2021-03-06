package amqputil

import (
	"context"

	"github.com/rinq/rinq-go/src/rinq/trace"
	"github.com/streadway/amqp"
)

// PackTrace sets msg.CorrelationId to traceID, only if it differs to msgID.
//
// The AMQP correlation ID field is used to tie "root" requests (be they command
// requests or notifications) to any requests that are made in response to that
// "root" request. This is different to the popular use of the correlation ID
// field, which is often used to relate a response to a request.
func PackTrace(msg *amqp.Publishing, traceID string) {
	if traceID != msg.MessageId {
		msg.CorrelationId = traceID
	}
}

// UnpackTrace creates a new context with a trace ID based on the AMQP correlation
// ID from msg.
//
// If the correlation ID is empty, the message is considered a "root" request,
// so the message ID is used as the correlation ID.
func UnpackTrace(parent context.Context, msg *amqp.Delivery) context.Context {
	if msg.CorrelationId != "" {
		return trace.With(parent, msg.CorrelationId)
	}

	return trace.With(parent, msg.MessageId)
}
