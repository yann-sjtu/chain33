package store

import (
	"context"

	"github.com/33cn/chain33/types"
	"github.com/gogo/protobuf/proto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type StoreHelper struct {
}

func (s *StoreHelper) PutBlock(routing *dht.IpfsDHT, block *types.Block, cfg *types.Chain33Config) error {
	key := MakeBlockHashAsKey(block.Hash(cfg))
	value, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return routing.PutValue(context.Background(), key, value)
}

func (s *StoreHelper) GetBlockByHash(routing *dht.IpfsDHT, hash []byte) (*types.Block, error) {
	key := MakeBlockHashAsKey(hash)
	value, err := routing.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}

	block := &types.Block{}
	err = proto.Unmarshal(value, block)
	if err != nil {
		return nil, err
	}

	return block, err
}
