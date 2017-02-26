package notifyamqp

import (
	"github.com/over-pass/overpass-go/src/internal/localsession"
	"github.com/over-pass/overpass-go/src/internal/notify"
	"github.com/over-pass/overpass-go/src/internal/revision"
	"github.com/over-pass/overpass-go/src/overpass"
	"github.com/over-pass/overpass-go/src/overpassamqp/internal/amqputil"
)

// New returns a pair of notifier and listener.
func New(
	peerID overpass.PeerID,
	config overpass.Config,
	sessions localsession.Store,
	revisions revision.Store,
	channels amqputil.ChannelPool,
) (notify.Notifier, notify.Listener, error) {
	channel, err := channels.Get() // do not return to pool, use for listener
	if err != nil {
		return nil, nil, err
	}

	if err = declareExchanges(channel); err != nil {
		return nil, nil, err
	}

	listener, err := newListener(
		peerID,
		config.SessionPreFetch,
		sessions,
		revisions,
		channel,
		config.Logger,
	)
	if err != nil {
		return nil, nil, err
	}

	return newNotifier(channels, config.Logger), listener, nil
}