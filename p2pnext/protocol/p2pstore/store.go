package p2pstore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"

	"github.com/33cn/chain33/common/log/log15"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
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

func genChunkPath(hash []byte) string {
	return fmt.Sprintf("/%s/%s", ChunkNameSpace, hex.EncodeToString(hash))
}

func genChunkKey(hash []byte) datastore.Key {
	return datastore.NewKey(genChunkPath(hash))
}
