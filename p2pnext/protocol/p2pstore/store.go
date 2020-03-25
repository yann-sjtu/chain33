package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/33cn/chain33/common/log/log15"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	"github.com/ipfs/go-datastore"
)

const (
	LocalChunkInfoKey = "local-chunk-info"
	ChunkNameSpace    = "chunk"
)

var (
	log        = log15.New("module", "protocol.p2pstore")
	AlphaValue = 3
	Backup     = 20
)

//StoreChunk handles notification of blockchain
// store chunk if this node is the nearest node in the local routing table
func (s *StoreProtocol) StoreChunk(req *types.ChunkInfo) error {
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertKey(genChunkPath(req.ChunkHash)), 1)
	if len(peers) != 0 && kbt.Closer(peers[0], s.Host.ID(), genChunkPath(req.ChunkHash)) {
		return nil
	}
	//如果p2pStore已保存数据，只更新时间即可
	if err := s.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := s.getChunkFromBlockchain(req)
	if err != nil {
		return err
	}
	data := types2.StorageData{
		Data:        bodys,
		RefreshTime: time.Now(),
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = s.DB.Put(genChunkKey(req.ChunkHash), b)
	if err != nil {
		return err
	}

	err = s.addLocalChunkInfo(req)
	if err != nil {
		return err
	}

	//本地存储之后立即到其他节点做依次备份
	s.notifyStoreChunk(req)
	return nil
}

func (s *StoreProtocol) updateChunk(req *types.ChunkInfo) error {
	b, err := s.DB.Get(genChunkKey(req.ChunkHash))
	if err != nil {
		return err
	}
	var data types2.StorageData
	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	if time.Since(data.RefreshTime) > types2.ExpiredTime {
		return types2.ErrDataExpired
	}
	data.RefreshTime = time.Now()
	b, err = json.Marshal(data)
	if err != nil {
		return err
	}
	return s.DB.Put(genChunkKey(req.ChunkHash), b)
}

func (s *StoreProtocol) GetChunk(req *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地数据
	data, err := s.DB.Get(genChunkKey(req.ChunkHash))
	if err == nil {
		var sData types2.StorageData
		err = json.Unmarshal(data, &sData)
		if err != nil {
			return nil, err
		}
		//本地数据没有过期则返回本地数据
		if time.Since(sData.RefreshTime) < types2.ExpiredTime {
			blocks := sData.Data.(*types.BlockBodys)
			if req.Filter {
				var bodyList []*types.BlockBody
				for _, body := range blocks.Items {
					if body.Height >= req.Start && body.Height <= req.End {
						bodyList = append(bodyList, body)
					}
				}
				blocks.Items = bodyList
			}
			return blocks, nil
		}
		//本地数据过期则删除本地数据
		err = s.deleteChunkBlock(req.ChunkHash)
		if err != nil {
			log.Error("GetChunk", "delete chunk error", err)
		}
	}

	//本地数据不存在或已过期，则向临近节点查询
	//首先从本地路由表获取 *3* 个最近的节点
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertKey(genChunkPath(req.ChunkHash)), AlphaValue)
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	for {
		bodys, newPeers := s.fetchChunkOrNearerPeersAsync(ctx, req, peers)
		if bodys != nil {
			return bodys, nil
		}
		if len(newPeers) == 0 {
			break
		}
		peers = newPeers
	}
	return nil, types2.ErrNotFound
}

func (s *StoreProtocol) deleteChunkBlock(hash []byte) error {
	err := s.deleteLocalChunkInfo(hash)
	if err != nil {
		return err
	}
	return s.DB.Delete(genChunkKey(hash))
}

func (s *StoreProtocol) addLocalChunkInfo(info *types.ChunkInfo) error {
	hashMap, err := s.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	hashMap[string(info.ChunkHash)] = info
	value, err := json.Marshal(hashMap)
	if err != nil {
		return err
	}

	err = s.DB.Put(datastore.NewKey(LocalChunkInfoKey), value)
	if err != nil {
		//索引存储失败，存储的区块也要回滚
		_ = s.DB.Delete(genChunkKey(info.ChunkHash))
	}
	return err
}

func (s *StoreProtocol) deleteLocalChunkInfo(hash []byte) error {
	hashMap, err := s.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	delete(hashMap, string(hash))
	value, err := json.Marshal(hashMap)
	if err != nil {
		return err
	}

	return s.DB.Put(datastore.NewKey(LocalChunkInfoKey), value)
}

func (s *StoreProtocol) getLocalChunkInfoMap() (map[string]*types.ChunkInfo, error) {

	ok, err := s.DB.Has(datastore.NewKey(LocalChunkInfoKey))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	value, err := s.DB.Get(datastore.NewKey(LocalChunkInfoKey))
	if err != nil {
		return nil, err
	}

	var chunkInfoMap map[string]*types.ChunkInfo
	err = json.Unmarshal(value, &chunkInfoMap)
	if err != nil {
		return nil, err
	}

	return chunkInfoMap, nil
}

func genChunkPath(hash []byte) string {
	return fmt.Sprintf("/%s/%s", ChunkNameSpace, hex.EncodeToString(hash))
}

func genChunkKey(hash []byte) datastore.Key {
	return datastore.NewKey(genChunkPath(hash))
}
