package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/33cn/chain33/common"
	protocol2 "github.com/33cn/chain33/p2pnext/protocol"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

const (
	LocalIndexHashListKey = "local-index-hash-list"
	BlocksPerPackage      = 1000
)

var AlphaValue = 3

var (
	ErrInvalidBlocksAmount = errors.New("invalid amount of blocks")
	ErrNotFound            = errors.New("not found")
	ErrInvalidHash         = errors.New("invalid hash")
	ErrCheckSum            = errors.New("check sum error")
)

func (s *StoreProtocol) SaveBlocks(blocks []*types.Block) error {
	//1000个区块打包一次
	if len(blocks) != BlocksPerPackage {
		return ErrInvalidBlocksAmount
	}
	hash := packageHash(blocks)
	//TODO 多次递归查询更大范围内最近de节点
	//TODO 目前返回20个，可以duo返回几个
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	peerCh, err := s.Discovery.Routing().GetClosestPeers(ctx, hash)
	if err != nil {
		return err
	}
	for peer := range peerCh {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
		stream, err := s.Host.NewStream(ctx, peer, PutData)
		if err != nil {
			//TODO +log
			continue
		}
		msg := protocol2.Message{
			ProtocolID: PutData,
			Params: types2.PutPackage{
				Hash:   hash,
				Blocks: blocks,
			},
		}
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		b, _ := json.Marshal(msg)
		rw.Write(b)
		rw.Flush()
		stream.Close()
	}
	return nil
}

func (s *StoreProtocol) GetBlocksByIndexHash(param *types2.FetchBlocks) ([]*types.Block, error) {
	if param == nil {
		return nil, ErrInvalidHash
	}

	b, err := s.DB.Get(datastore.NewKey(param.Hash))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrNotFound
	}

	//本地不存在，则向临近节点查询
	//首先从本地路由表获取 *3* 个最近的节点
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertKey(param.Hash), 3)
	responseCh := make(chan protocol2.Response, AlphaValue)
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	//递归查询，直到查询到数据
	//TODO 加超时时间
Iter:
	for _, peerID := range peers {
		go s.getBlocksFromRemote(cancelCtx, param, peerID, responseCh)
	}

	for res := range responseCh {
		//三个并发请求任意一个正常返回时，cancel掉另外两个
		if res.Error != nil {
			continue
		}
		cancelFunc()
		if blocks, ok := res.Result.([]*types.Block); ok {
			return blocks, nil
		}
		if newPeers, ok := res.Result.([]peer.ID); ok {
			peers = newPeers
			responseCh = make(chan protocol2.Response, AlphaValue)
			cancelCtx, cancelFunc = context.WithCancel(context.Background())
			goto Iter
		}
	}
	//不应该执行到这里
	//debug
	panic(err)
	//debug
	return nil, ErrNotFound
}

func (s *StoreProtocol) Republish() error {
	hashMap, err := s.GetLocalIndexHash()
	if err != nil {
		return err
	}

	for hash := range hashMap {
		value, err := s.DB.Get(datastore.NewKey(hash))
		if err != nil {
			//TODO +log
			continue
		}
		var blocks []*types.Block
		err = json.Unmarshal(value, &blocks)
		if err != nil {
			//TODO +log
			continue
		}
		err = s.SaveBlocks(blocks)
		if err != nil {
			//TODO +log
			continue
		}
	}

	return nil
}

func (s *StoreProtocol) AddLocalIndexHash(hash string) error {
	hashMap, err := s.GetLocalIndexHash()
	if err != nil {
		return err
	}

	hashMap[hash] = struct{}{}
	value, err := json.Marshal(hashMap)
	if err != nil {
		return err
	}

	return s.DB.Put(datastore.NewKey(LocalIndexHashListKey), value)
}

func (s *StoreProtocol) DeleteLocalIndexHash(hash string) error {
	hashMap, err := s.GetLocalIndexHash()
	if err != nil {
		return err
	}

	delete(hashMap, hash)
	value, err := json.Marshal(hashMap)
	if err != nil {
		return err
	}

	return s.DB.Put(datastore.NewKey(LocalIndexHashListKey), value)
}

func (s *StoreProtocol) GetLocalIndexHash() (map[string]struct{}, error) {
	value, err := s.DB.Get(datastore.NewKey(LocalIndexHashListKey))
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	var hashMap map[string]struct{}
	err = json.Unmarshal(value, &hashMap)
	if err != nil {
		return nil, err
	}

	return hashMap, nil
}

//packageHash 计算归档哈希,具体计算方式后续可能修改
func packageHash(blocks []*types.Block) string {
	value, err := json.Marshal(blocks)
	if err != nil {
		panic(err)
	}
	return string(common.Sha256(value))
}
