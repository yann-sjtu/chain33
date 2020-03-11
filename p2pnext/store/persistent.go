package store

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	recpb "github.com/libp2p/go-libp2p-record/pb"
	"github.com/33cn/chain33/types"
)

type Persistent struct {
	//for test
	Values map[datastore.Key][]byte
}

func (p *Persistent) Get(key datastore.Key) (value []byte, err error) {
	val, found := p.Values[key]
	if !found {
		return nil, nil
	}
	return val, nil
}

func (p *Persistent) Has(key datastore.Key) (exists bool, err error) {
	_, found := p.Values[key]
	return found, nil
}

func (p *Persistent) GetSize(key datastore.Key) (size int, err error) {
	return -1, errors.New("GetSize ErrNotFound")
}

func (p *Persistent) Query(q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (p *Persistent) Put(key datastore.Key, value []byte) error {
	p.Values[key] = value

	rec := &recpb.Record{}
	err := proto.Unmarshal(value, rec)
	if err != nil {
		return errors.New("Put Unmarshal error ")
	}

	//todo:as block unmarshal,should support unmarshal by type
	block:=&types.Block{}
	err = proto.Unmarshal(rec.Value, block)
	if err != nil {
		return  err
	}
	fmt.Println("kadtest block height ",block.Height)

	return nil
}

func (p *Persistent) Delete(key datastore.Key) error {
	return nil
}

func (p *Persistent) Close() error {
	return nil
}

func (p *Persistent) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(p), nil
}
