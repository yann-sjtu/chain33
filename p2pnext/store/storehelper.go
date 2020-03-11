package store

import (
	"context"
	"errors"

	"github.com/33cn/chain33/types"
	"github.com/gogo/protobuf/proto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type StoreHelper struct {
}

func (bs *StoreHelper) PutBlock(routing *dht.IpfsDHT, block *types.Block, cfg *types.Chain33Config) error {
	key := MakeBlockHashAsKey(block.Hash(cfg))
	value, err := MakeBlockAsValue(block)
	if err != nil {
		return errors.New("PutBlock  MakeBlockAsValue error ")
	}

	err = routing.PutValue(context.Background(), key, value)

	return err
}

func (bs *StoreHelper) GetBlockByHash(routing *dht.IpfsDHT, hash []byte) (*types.Block, error) {
	key := MakeBlockHashAsKey(hash)
	value, err := routing.GetValue(context.Background(), key)
	if err != nil {
		return nil, errors.New("GetBlockByHash GetValue error ")
	}

	var block *types.Block
	err = proto.Unmarshal(value, block)

	return block, err
}
