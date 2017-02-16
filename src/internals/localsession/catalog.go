package localsession

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/over-pass/overpass-go/src/internals"
	"github.com/over-pass/overpass-go/src/internals/deferutil"
	"github.com/over-pass/overpass-go/src/overpass"
)

// Catalog is an interface for manipulating an attribute table.
// There is a one-to-one relationship between sessions and catalogs.
type Catalog interface {
	// Ref returns the most recent session-ref.
	// The ref's revision increments each time a call to TryUpdate() succeeds.
	Ref() overpass.SessionRef

	// NextMessageID generates a unique message ID from the current session-ref.
	NextMessageID() overpass.MessageID

	// Head returns the most recent revision.
	// It is conceptually equivalent to catalog.At(catalog.Ref().Rev).
	Head() overpass.Revision

	// At returns a revision representing the catalog at a specific revision
	// number. The revision can not be newer than the current session-ref.
	At(overpass.RevisionNumber) (overpass.Revision, error)

	// Attrs returns all attributes at the most recent revision.
	Attrs() (overpass.SessionRef, internals.AttrTableWithMetaData)

	// TryUpdate adds or updates attributes in the attribute table and returns
	// the new head revision.
	//
	// The operation fails if ref is not the current session-ref, attrs includes
	// changes to frozen attributes, or the catalog is closed.
	//
	// A human-readable representation of the changes is written to diff, if it
	// is non-nil.
	TryUpdate(
		ref overpass.SessionRef,
		attrs []overpass.Attr,
		diff io.Writer,
	) (overpass.Revision, error)

	// TryClose closes the catalog, preventing further updates.
	//
	// The operation fails if ref is not the current session-ref. It is not an
	// error to close an already-closed catalog.
	//
	// The return value is only true if this call caused the catalog to close.
	TryClose(ref overpass.SessionRef) (bool, error)

	// Close forcefully closes the catalog, preventing further updates.
	// It is not an error to close an already-closed catalog.
	Close()
}

type catalog struct {
	mutex    sync.RWMutex
	ref      overpass.SessionRef
	attrs    internals.AttrTableWithMetaData
	seq      uint32
	isClosed bool
	logger   *log.Logger
}

// NewCatalog returns a catalog for the given session.
func NewCatalog(
	id overpass.SessionID,
	logger *log.Logger,
) Catalog {
	return &catalog{
		ref:    id.At(0),
		logger: logger,
	}
}

func (c *catalog) Ref() overpass.SessionRef {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ref
}

func (c *catalog) NextMessageID() overpass.MessageID {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.seq++
	return overpass.MessageID{Session: c.ref, Seq: c.seq}
}

func (c *catalog) Head() overpass.Revision {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return &revision{
		ref:     c.ref,
		catalog: c,
		attrs:   c.attrs,
		logger:  c.logger,
	}
}

func (c *catalog) At(rev overpass.RevisionNumber) (overpass.Revision, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.ref.Rev < rev {
		return nil, errors.New("revision is from the future")
	}

	return &revision{
		ref:     c.ref.ID.At(rev),
		catalog: c,
		attrs:   c.attrs,
		logger:  c.logger,
	}, nil
}

func (c *catalog) Attrs() (overpass.SessionRef, internals.AttrTableWithMetaData) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.ref, c.attrs
}

func (c *catalog) TryUpdate(
	ref overpass.SessionRef,
	attrs []overpass.Attr,
	diff io.Writer,
) (overpass.Revision, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isClosed {
		return nil, overpass.NotFoundError{ID: c.ref.ID}
	}

	if ref != c.ref {
		return nil, overpass.StaleUpdateError{Ref: ref}
	}

	nextAttrs := c.attrs.Clone()
	nextRev := ref.Rev + 1

	var frozen []string

	for index, attr := range attrs {
		entry, exists := nextAttrs[attr.Key]

		if attr.Value == entry.Value && attr.IsFrozen == entry.IsFrozen {
			continue
		}

		if entry.IsFrozen {
			frozen = append(frozen, attr.Key)
			continue
		}

		if !exists {
			entry.Key = attr.Key
			entry.CreatedAt = nextRev
		}

		entry.Value = attr.Value
		entry.UpdatedAt = nextRev

		nextAttrs[attr.Key] = entry

		if diff != nil {
			if index > 0 {
				_, err := io.WriteString(diff, ", ")
				if err != nil {
					return nil, err
				}
			}
			if err := writeDiff(diff, entry); err != nil {
				return nil, err
			}
		}
	}

	if len(frozen) > 0 {
		return nil, overpass.FrozenAttributesError{Ref: ref, Keys: frozen}
	}

	c.ref.Rev = nextRev
	c.attrs = nextAttrs
	c.seq = 0

	return &revision{
		ref:     c.ref,
		catalog: c,
		attrs:   c.attrs,
	}, nil
}

func (c *catalog) TryClose(ref overpass.SessionRef) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ref != c.ref {
		return false, overpass.StaleUpdateError{Ref: ref}
	}

	if c.isClosed {
		return false, nil
	}

	c.isClosed = true
	return true, nil
}

func (c *catalog) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.isClosed = true
}

func writeDiff(w io.Writer, attr internals.AttrWithMetaData) (err error) {
	defer deferutil.Recover(&err)

	write := func(s string) {
		if _, e := io.WriteString(w, s); e != nil {
			panic(e)
		}
	}

	if attr.Value == "" {
		if attr.IsFrozen {
			write("!")
		} else {
			write("-")
		}

		write(attr.Key)
	} else {
		if attr.CreatedAt == attr.UpdatedAt {
			write("+")
		}

		write(attr.Key)

		if attr.IsFrozen {
			write("@")
		} else {
			write("=")
		}

		write(attr.Value)
	}

	return
}