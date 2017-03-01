package revision

import (
	"context"

	"github.com/rinq/rinq-go/src/rinq"
)

// Closed returns an revision that behaves as though its session has
// been closed.
func Closed(ref rinq.SessionRef) rinq.Revision {
	return closedRevision(ref)
}

type closedRevision rinq.SessionRef

func (r closedRevision) Ref() rinq.SessionRef {
	return rinq.SessionRef(r)
}

func (r closedRevision) Refresh(context.Context) (rinq.Revision, error) {
	return nil, rinq.NotFoundError{ID: r.ID}
}

func (r closedRevision) Get(context.Context, string) (rinq.Attr, error) {
	return rinq.Attr{}, rinq.NotFoundError{ID: r.ID}
}

func (r closedRevision) GetMany(context.Context, ...string) (rinq.AttrTable, error) {
	return nil, rinq.NotFoundError{ID: r.ID}
}

func (r closedRevision) Update(context.Context, ...rinq.Attr) (rinq.Revision, error) {
	return r, rinq.NotFoundError{ID: r.ID}
}

func (r closedRevision) Close(context.Context) error {
	return nil
}