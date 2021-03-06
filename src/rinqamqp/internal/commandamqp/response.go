package commandamqp

import (
	"context"
	"fmt"
	"sync"

	"github.com/rinq/rinq-go/src/rinq"
	"github.com/rinq/rinq-go/src/rinq/trace"
	"github.com/rinq/rinq-go/src/rinqamqp/internal/amqputil"
	"github.com/streadway/amqp"
)

// response is used to send responses to command requests, it implements
// rinq.Response.
type response struct {
	context  context.Context
	channels amqputil.ChannelPool
	request  rinq.Request

	mutex     sync.RWMutex
	replyMode replyMode
	isClosed  bool
}

func newResponse(
	ctx context.Context,
	channels amqputil.ChannelPool,
	request rinq.Request,
	replyMode replyMode,
) (rinq.Response, func() bool) {
	r := &response{
		context:   ctx,
		channels:  channels,
		request:   request,
		replyMode: replyMode,
	}

	return r, r.finalize
}

func (r *response) IsRequired() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.isClosed {
		return false
	}

	if r.replyMode == replyNone {
		return false
	}

	select {
	case <-r.context.Done():
		return false
	default:
		return true
	}
}

func (r *response) IsClosed() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.isClosed
}

func (r *response) Done(payload *rinq.Payload) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isClosed {
		panic("responder is already closed")
	}

	msg := &amqp.Publishing{}
	packSuccessResponse(msg, payload)
	r.respond(msg)
}

func (r *response) Error(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isClosed {
		panic("responder is already closed")
	}

	msg := &amqp.Publishing{}
	packErrorResponse(msg, err)
	r.respond(msg)
}

func (r *response) Fail(t, f string, v ...interface{}) rinq.Failure {
	err := rinq.Failure{
		Type:    t,
		Message: fmt.Sprintf(f, v...),
	}

	r.Error(err)

	return err
}

func (r *response) Close() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isClosed {
		return false
	}

	msg := &amqp.Publishing{}
	packSuccessResponse(msg, nil)
	r.respond(msg)

	return true
}

func (r *response) finalize() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isClosed {
		return true
	}

	r.isClosed = true

	return false
}

func (r *response) respond(msg *amqp.Publishing) {
	r.isClosed = true

	if r.replyMode == replyNone {
		return
	}

	if _, err := amqputil.PackDeadline(r.context, msg); err != nil {
		// the context deadline has already passed
		return
	}

	channel, err := r.channels.Get()
	if err != nil {
		panic(err)
	}
	defer r.channels.Put(channel)

	// TODO: is this necessary for correlated responses?
	amqputil.PackTrace(msg, trace.Get(r.context))

	if r.replyMode == replyUncorrelated {
		packNamespaceAndCommand(msg, r.request.Namespace, r.request.Command)
		packReplyMode(msg, r.replyMode)

		err = amqputil.PackSpanContext(r.context, msg)
		if err != nil {
			panic(err)
		}
	}

	err = channel.Publish(
		responseExchange,
		r.request.ID.String(),
		false, // mandatory,
		false, // immediate,
		*msg,
	)
	if err != nil {
		panic(err)
	}
}
