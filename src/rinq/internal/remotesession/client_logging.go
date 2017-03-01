package remotesession

import (
	"context"

	"github.com/rinq/rinq-go/src/rinq"
	"github.com/rinq/rinq-go/src/rinq/internal/attrmeta"
	"github.com/rinq/rinq-go/src/rinq/internal/bufferpool"
	"github.com/rinq/rinq-go/src/rinq/internal/trace"
)

func logUpdate(
	ctx context.Context,
	logger rinq.Logger,
	peerID rinq.PeerID,
	ref rinq.SessionRef,
	attrs []attrmeta.Attr,
) {
	buffer := bufferpool.Get()
	defer bufferpool.Put(buffer)

	for _, attr := range attrs {
		if buffer.Len() != 0 {
			buffer.WriteString(", ")
		}

		attrmeta.Write(buffer, attr)
	}

	logger.Log(
		"%s updated remote session %s {%s} [%s]",
		peerID.ShortString(),
		ref.ShortString(),
		buffer.String(),
		trace.Get(ctx),
	)
}

func logClose(
	ctx context.Context,
	logger rinq.Logger,
	peerID rinq.PeerID,
	ref rinq.SessionRef,
) {
	logger.Log(
		"%s destroyed remote session %s [%s]",
		peerID.ShortString(),
		ref.ShortString(),
		trace.Get(ctx),
	)
}