package localsession

import (
	"errors"
	"sync"

	"github.com/rinq/rinq-go/src/rinq"
	"github.com/rinq/rinq-go/src/rinq/ident"
	"github.com/rinq/rinq-go/src/rinq/internal/attributes"
	"github.com/rinq/rinq-go/src/rinq/internal/attrmeta"
)

// Catalog is an interface for manipulating an attribute table.
// There is a one-to-one relationship between sessions and catalogs.
type Catalog interface {
	// Ref returns the most recent session-ref.
	// The ref's revision increments each time a call to TryUpdate() succeeds.
	Ref() ident.Ref

	// NextMessageID generates a unique message ID from the current session-ref.
	NextMessageID() (ident.MessageID, attrmeta.Table)

	// Head returns the most recent revision.
	// It is conceptually equivalent to catalog.At(catalog.Ref().Rev).
	Head() rinq.Revision

	// At returns a revision representing the catalog at a specific revision
	// number. The revision can not be newer than the current session-ref.
	At(ident.Revision) (rinq.Revision, error)

	// Attrs returns all attributes at the most recent revision.
	Attrs() (ident.Ref, attrmeta.Table)

	// AttrsIn returns all attributes in the ns namespace at the most recent revision.
	AttrsIn(ns string) (ident.Ref, attrmeta.Namespace)

	// TryUpdate adds or updates attributes in the ns namespace of the attribute
	// table and returns the new head revision.
	//
	// The operation fails if ref is not the current session-ref, attrs includes
	// changes to frozen attributes, or the catalog is closed.
	TryUpdate(
		ref ident.Ref,
		ns string,
		attrs attributes.List,
	) (rinq.Revision, *attrmeta.Diff, error)

	// TryClear updates all attributes in the ns namespace of the attribute
	// table to the empty string and returns the new head revision.
	//
	// The operation fails if ref is not the current session-ref, there are any
	// frozen attributes, or the catalog is closed.
	TryClear(
		ref ident.Ref,
		ns string,
	) (rinq.Revision, *attrmeta.Diff, error)

	// TryDestroy closes the catalog, preventing further updates.
	//
	// The operation fails if ref is not the current session-ref. It is not an
	// error to close an already-closed catalog.
	TryDestroy(ref ident.Ref) error

	// Close forcefully closes the catalog, preventing further updates.
	// It is not an error to close an already-closed catalog.
	Close()

	// Done returns a channel that is closed when the catalog is closed.
	Done() <-chan struct{}
}

type catalog struct {
	mutex  sync.RWMutex
	ref    ident.Ref
	attrs  attrmeta.Table
	seq    uint32
	done   chan struct{}
	logger rinq.Logger
}

// NewCatalog returns a catalog for the given session.
func NewCatalog(
	id ident.SessionID,
	logger rinq.Logger,
) Catalog {
	return &catalog{
		ref:    id.At(0),
		done:   make(chan struct{}),
		logger: logger,
	}
}

func (c *catalog) Ref() ident.Ref {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ref
}

func (c *catalog) NextMessageID() (ident.MessageID, attrmeta.Table) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.seq++
	return c.ref.Message(c.seq), c.attrs
}

func (c *catalog) Head() rinq.Revision {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return &revision{c.ref, c, c.attrs, c.logger}
}

func (c *catalog) At(rev ident.Revision) (rinq.Revision, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.ref.Rev < rev {
		return nil, errors.New("revision is from the future")
	}

	return &revision{
		c.ref.ID.At(rev),
		c,
		c.attrs,
		c.logger,
	}, nil
}

func (c *catalog) Attrs() (ident.Ref, attrmeta.Table) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ref, c.attrs
}

func (c *catalog) AttrsIn(ns string) (ident.Ref, attrmeta.Namespace) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ref, c.attrs[ns]
}

func (c *catalog) TryUpdate(
	ref ident.Ref,
	ns string,
	attrs attributes.List,
) (rinq.Revision, *attrmeta.Diff, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case <-c.done:
		return nil, nil, rinq.NotFoundError{ID: c.ref.ID}
	default:
	}

	if ref != c.ref {
		return nil, nil, rinq.StaleUpdateError{Ref: ref}
	}

	nextRev := ref.Rev + 1
	nextAttrs := c.attrs[ns].Clone()
	diff := attrmeta.NewDiff(ns, nextRev, len(attrs))

	for _, attr := range attrs {
		entry, exists := nextAttrs[attr.Key]

		if attr.Value == entry.Value && attr.IsFrozen == entry.IsFrozen {
			continue
		}

		if entry.IsFrozen {
			return nil, nil, rinq.FrozenAttributesError{Ref: ref}
		}

		entry.Attr = attr
		entry.UpdatedAt = nextRev
		if !exists {
			entry.CreatedAt = nextRev
		}

		nextAttrs[attr.Key] = entry
		diff.Append(entry)
	}

	c.ref.Rev = nextRev
	c.seq = 0

	if !diff.IsEmpty() {
		c.attrs = c.attrs.CloneAndMerge(ns, nextAttrs)
	}

	return &revision{
		c.ref,
		c,
		c.attrs,
		c.logger,
	}, diff, nil
}

func (c *catalog) TryClear(
	ref ident.Ref,
	ns string,
) (rinq.Revision, *attrmeta.Diff, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case <-c.done:
		return nil, nil, rinq.NotFoundError{ID: c.ref.ID}
	default:
	}

	if ref != c.ref {
		return nil, nil, rinq.StaleUpdateError{Ref: ref}
	}

	attrs := c.attrs[ns]
	nextRev := ref.Rev + 1
	nextAttrs := attrmeta.Namespace{}
	diff := attrmeta.NewDiff(ns, nextRev, len(nextAttrs))

	for _, entry := range attrs {
		if entry.Value != "" {
			if entry.IsFrozen {
				return nil, nil, rinq.FrozenAttributesError{Ref: ref}
			}

			entry.Value = ""
			entry.UpdatedAt = nextRev
			diff.Append(entry)
		}

		nextAttrs[entry.Key] = entry
	}

	c.ref.Rev = nextRev
	c.seq = 0

	if !diff.IsEmpty() {
		c.attrs = c.attrs.CloneAndMerge(ns, nextAttrs)
	}

	return &revision{
		c.ref,
		c,
		c.attrs,
		c.logger,
	}, diff, nil
}

func (c *catalog) TryDestroy(ref ident.Ref) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ref != c.ref {
		return rinq.StaleUpdateError{Ref: ref}
	}

	select {
	case <-c.done:
	default:
		close(c.done)
	}

	return nil
}

func (c *catalog) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *catalog) Done() <-chan struct{} {
	return c.done
}
