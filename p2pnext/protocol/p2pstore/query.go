package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"time"

	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (s *StoreProtocol) getHeaders(param *types.ReqBlockHeaders) []*types.Header {
	for _, pid := range s.Discovery.RoutingTale() {
		childCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		stream, err := s.Host.NewStream(childCtx, pid, GetHeader)
		if err != nil {
			continue
		}
		msg := types2.Message{
			ProtocolID: GetHeader,
			Params:     param,
		}
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		b, _ := json.Marshal(msg)
		rw.Write(b)
		rw.Flush()
		res, err := readResponse(stream)
		stream.Close()
		if err != nil {
			continue
		}
		headers := res.Result.([]*types.Header)
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return nil
}

func (s *StoreProtocol) getChunkRecords(param *types.ReqChunkRecords) *types.ChunkRecords {
	for _, pid := range s.Discovery.RoutingTale() {
		childCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		stream, err := s.Host.NewStream(childCtx, pid, GetChunkRecord)
		if err != nil {
			continue
		}
		msg := types2.Message{
			ProtocolID: GetChunkRecord,
			Params:     param,
		}
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		b, _ := json.Marshal(msg)
		rw.Write(b)
		rw.Flush()
		res, err := readResponse(stream)
		stream.Close()
		if err != nil {
			continue
		}
		records := res.Result.(*types.ChunkRecords)
		return records
	}

	log.Error("getChunkRecords", "error", types2.ErrNotFound)
	return nil
}

// fetchChunkOrNearerPeersAsync 返回 *types.ChunkBlockBody 或者 []peer.ID
func (s *StoreProtocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ReqChunkBlockBody, peers []peer.ID) interface{} {

	responseCh := make(chan *types2.Response, AlphaValue)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	// *3* 个节点并发请求
	for _, peerID := range peers {
		go func(pid peer.ID) {
			responseCh <- s.fetchChunkOrNearerPeers(cancelCtx, param, pid)
		}(peerID)
	}

	var peerList []peer.ID
	for res := range responseCh {
		if res == nil || res.Error != nil {
			continue
		}
		//没查到区块数据，返回了更近的节点信息，继续迭代查询
		if newPeers, ok := res.Result.([]peer.ID); ok {
			peerList = append(peerList, newPeers...)
			continue
		}
		//查到了区块数据，直接返回
		if bodys, ok := res.Result.(*types.BlockBodys); ok {
			return bodys
		}

		//返回类型不是*types.BlockBodys或[]peer.ID，对端节点异常
		log.Error("fetchChunkAsync", "fetchChunkOrNearerPeers invalid response", res.Result)
	}

	//TODO 若超过3个，排序选择最优的三个节点
	if len(peerList) > AlphaValue {
		peerList = peerList[:AlphaValue]
	}

	return peerList
}

func (s *StoreProtocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ReqChunkBlockBody, pid peer.ID) *types2.Response {
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, FetchChunk)
	if err != nil {
		log.Error("getBlocksFromRemote", "error", err)
		return nil
	}
	defer stream.Close()
	msg := types2.Message{
		ProtocolID: FetchChunk,
		Params:     params,
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	b, _ := json.Marshal(msg)
	rw.Write(b)
	rw.Flush()
	res, err := readResponse(stream)
	if err != nil {
		log.Error("fetchChunkFromPeer", "read response error", err)
		return nil
	}
	log.Info("fetchChunkOrNearerPeers response ok", "remote peer", stream.Conn().RemotePeer().Pretty())

	return res
}

func readResponse(stream network.Stream) (*types2.Response, error) {
	var data []byte
	var err error
	for {
		buf := make([]byte, 100)
		var n int
		n, err = stream.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		data = append(data, buf[:n]...)
		if err == io.EOF {
			break
		}
	}

	var res types2.Response
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func readMessage(stream network.Stream) (*types2.Message, error) {
	var data []byte
	for {
		buf := make([]byte, 100)
		var n int
		n, err := stream.Read(buf)
		if err != nil && err != io.EOF {
			log.Error("Handle", "read stream error", err)
			return nil, err
		}
		data = append(data, buf[:n]...)
		if err == io.EOF {
			break
		}
	}

	var msg types2.Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
