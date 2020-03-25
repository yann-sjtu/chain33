package p2pstore

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	kbt "github.com/libp2p/go-libp2p-kbucket"

	types2 "github.com/33cn/chain33/p2pnext/types"
)

func (s *StoreProtocol) startRepublish() {
	for range time.Tick(types2.RefreshInterval) {
		if err := s.republish(); err != nil {
			log.Error("cycling republish", "error", err)
		}
	}
}

func (s *StoreProtocol) republish() error {
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
		s.notifyStoreChunk(info)
	}

	return nil
}

func (s *StoreProtocol) notifyStoreChunk(req *types.ChunkInfo) {
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertKey(genChunkPath(req.ChunkHash)), Backup)
	for _, pid := range peers {
		err := s.storeChunkOnPeer(req, pid)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
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
