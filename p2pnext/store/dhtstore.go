package p2pnext

import (
	"errors"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

type DHTStore struct {
	//for test
	Values map[datastore.Key][]byte
}

func (d *DHTStore) Get(key datastore.Key) (value []byte, err error) {
	val, found := d.Values[key]
	if !found {
		return nil, nil
	}
	return val, nil
}

func (d *DHTStore) Has(key datastore.Key) (exists bool, err error) {
	_, found := d.Values[key]
	return found, nil
}

func (d *DHTStore) GetSize(key datastore.Key) (size int, err error) {
	return -1, errors.New("GetSize ErrNotFound")
}

func (d *DHTStore) Query(q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (d *DHTStore) Put(key datastore.Key, value []byte) error {
	d.Values[key] =value
	return nil
}

func (d *DHTStore) Delete(key datastore.Key) error {
	return nil
}

func (d *DHTStore) Close() error {
	return nil
}

func (d *DHTStore) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(d), nil
}


