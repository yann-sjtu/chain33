package p2pstore

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
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
)

func (s *StoreProtocol) StoreChunk(req *types.ChunkInfo) error {
	//TODO 多次递归查询更大范围内最近的节点
	//TODO 目前返回20个，可以多返回几个
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	peerCh, err := s.Discovery.Routing().GetClosestPeers(ctx, genChunkPath(req.ChunkHash))
	if err != nil {
		return err
	}
	for pid := range peerCh {
		err = s.storeChunkOnPeer(req, pid)
		if err != nil {
			log.Error("new stream error when store chunk", "peer id", pid, "error", err)
			continue
		}
	}
	return nil
}

func (s *StoreProtocol) storeChunkOnPeer(req *types.ChunkInfo, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	stream, err := s.Host.NewStream(ctx, pid, StoreChunk)
	if err != nil {
		log.Error("new stream error when store chunk", "peer id", pid, "error", err)
		return err
	}
	msg := types2.Message{
		ProtocolID: StoreChunk,
		Params:     req,
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = rw.Write(b)
	if err != nil {
		return err
	}
	err = rw.Flush()
	if err != nil {
		return err
	}
	err = stream.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreProtocol) GetChunk(param *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if param == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地数据
	data, err := s.DB.Get(genChunkKey(param.ChunkHash))
	if err == nil {
		var sData types2.StorageData
		err = json.Unmarshal(data, &sData)
		if err != nil {
			return nil, err
		}
		//本地数据没有过期则返回本地数据
		if time.Since(sData.RefreshTime) < types2.ExpiredTime {
			blocks := sData.Data.(*types.BlockBodys)
			if param.Filter {
				var bodyList []*types.BlockBody
				for _, body := range blocks.Items {
					if body.Height >= param.Start && body.Height <= param.End {
						bodyList = append(bodyList, body)
					}
				}
				blocks.Items = bodyList
			}
			return blocks, nil
		}
		//本地数据过期则删除本地数据
		err = s.deleteChunkBlock(param.ChunkHash)
		if err != nil {
			log.Error("GetChunk", "delete chunk error", err)
		}
	}

	//本地数据不存在或已过期，则向临近节点查询
	//首先从本地路由表获取 *3* 个最近的节点
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertKey(genChunkPath(param.ChunkHash)), AlphaValue)
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	for {
		bodys, newPeers := s.fetchChunkOrNearerPeersAsync(ctx, param, peers)
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

	return s.DB.Put(datastore.NewKey(LocalChunkInfoKey), value)
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
