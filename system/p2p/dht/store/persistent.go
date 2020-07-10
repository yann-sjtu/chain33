package store

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

// Persistent implements datastore.
type Persistent struct {
	db dbm.DB
}

// Get gets data from db.
func (p *Persistent) Get(key datastore.Key) (value []byte, err error) {
	v, e := p.db.Get(key.Bytes())
	if v == nil {
		return nil, datastore.ErrNotFound
	}
	return v, e
}

// Has returns true if value is not nil.
func (p *Persistent) Has(key datastore.Key) (exists bool, err error) {
	value, err := p.db.Get(key.Bytes())
	return value != nil, err
}

// GetSize returns size of data.
func (p *Persistent) GetSize(key datastore.Key) (size int, err error) {
	v, err := p.Get(key)
	if err != nil {
		return -1, datastore.ErrNotFound
	}
	return len(v), nil
}

// Query do not use
func (p *Persistent) Query(q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

// Put saves data.
func (p *Persistent) Put(key datastore.Key, value []byte) error {
	return p.db.Set(key.Bytes(), value)
}

// Delete deletes data.
func (p *Persistent) Delete(key datastore.Key) error {
	return p.db.Delete(key.Bytes())
}

// Close closes db.
func (p *Persistent) Close() error {
	p.db.Close()
	return nil
}

// Batch creates a batch object.
func (p *Persistent) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(p), nil
}
