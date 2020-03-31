package p2pstore

import (
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
	Backup     = 20
)

// 保存chunk到本地p2pStore，同时更新本地chunk列表
func (s *StoreProtocol) addChunkBlock(info *types.ChunkInfo, bodys *types.BlockBodys) error {
	b, err := json.Marshal(types2.StorageData{
		Data:        bodys,
		RefreshTime: time.Now(),
	})
	if err != nil {
		return err
	}
	err = s.addLocalChunkInfo(info)
	if err != nil {
		return err
	}
	return s.DB.Put(genChunkKey(info.ChunkHash), b)
}

// 更新本地chunk保存时间，chunk不存在则返回error
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
	data.RefreshTime = time.Now()
	b, err = json.Marshal(data)
	if err != nil {
		return err
	}
	return s.DB.Put(genChunkKey(req.ChunkHash), b)
}

// 获取本地chunk数据，若数据已过期则删除该数据并返回空
func (s *StoreProtocol) getChunkBlock(hash []byte) (*types.BlockBodys, error) {
	b, err := s.DB.Get(genChunkKey(hash))
	if err != nil {
		return nil, err
	}
	var data types2.StorageData
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}
	if time.Since(data.RefreshTime) > types2.ExpiredTime {
		err = s.DB.Delete(genChunkKey(hash))
		if err != nil {
			log.Error("getChunkBlock", "delete chunk error", err, "hash", hex.EncodeToString(hash))
			return nil, err
		}
		return nil, types2.ErrNotFound
	}

	return data.Data.(*types.BlockBodys), nil

}

func (s *StoreProtocol) deleteChunkBlock(hash []byte) error {
	err := s.deleteLocalChunkInfo(hash)
	if err != nil {
		return err
	}
	return s.DB.Delete(genChunkKey(hash))
}

// 保存一个本地chunk hash列表，用于遍历本地数据
func (s *StoreProtocol) addLocalChunkInfo(info *types.ChunkInfo) error {
	hashMap, err := s.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	if _, ok := hashMap[string(info.ChunkHash)]; ok {
		return nil
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

// 适配libp2p，按路径格式生成数据的key值，便于区分多种数据类型的命名空间，以及key值合法性校验
func genChunkPath(hash []byte) string {
	return fmt.Sprintf("/%s/%s", ChunkNameSpace, hex.EncodeToString(hash))
}

func genChunkKey(hash []byte) datastore.Key {
	return datastore.NewKey(genChunkPath(hash))
}
