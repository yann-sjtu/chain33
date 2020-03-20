package p2pstore

import (
	"encoding/hex"
	"encoding/json"
	"time"

	types2 "github.com/33cn/chain33/p2pnext/types"
)

func (s *StoreProtocol) Republish() error {
	chunkInfoMap, err := s.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	for hash, info := range chunkInfoMap {
		b, err := s.DB.Get(genChunkKey([]byte(hash)))
		if err != nil {
			log.Error("republish get error", "hash", hex.EncodeToString([]byte(hash)), "error", err)
			continue
		}
		var data types2.StorageData
		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Error("republish unmarshal error", "hash", hex.EncodeToString([]byte(hash)), "error", err)
			continue
		}
		if time.Since(data.RefreshTime) > types2.ExpiredTime {
			err = s.deleteChunkBlock([]byte(hash))
			if err != nil {
				log.Error("Republish", "delete chunk error", err)
			}
			continue
		}
		err = s.StoreChunk(info)
		if err != nil {
			log.Error("republish store chunk error", "hash", hex.EncodeToString([]byte(hash)), "error", err)
			continue
		}
	}

	return nil
}
