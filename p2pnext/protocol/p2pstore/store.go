package p2pstore

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

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
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	peerCh, err := s.Discovery.Routing().GetClosestPeers(ctx, genChunkPath(req.ChunkHash))
	if err != nil {
		return err
	}
	for pid := range peerCh {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
		stream, err := s.Host.NewStream(ctx, pid, StoreChunk)
		if err != nil {
			log.Error("new stream error when store chunk", "peer id", pid, "error", err)
			continue
		}
		msg := types2.Message{
			ProtocolID: StoreChunk,
			Params:     req,
		}
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		b, _ := json.Marshal(msg)
		rw.Write(b)
		rw.Flush()
		stream.Close()
	}
	return nil
}

func (s *StoreProtocol) GetChunk(param *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if param == nil {
		return nil, types2.ErrInvalidParam
	}

	data, err := s.DB.Get(genChunkKey(param.ChunkHash))
	if err == nil {
		var sData types2.StorageData
		err = json.Unmarshal(data, &sData)
		if err != nil {
			return nil, err
		}
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
		err = s.deleteChunkBlock(param.ChunkHash)
		if err != nil {
			log.Error("GetChunk", "delete chunk error", err)
		}
	}

	//本地不存在，则向临近节点查询
	return s.fetchChunkAsync(param)
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
