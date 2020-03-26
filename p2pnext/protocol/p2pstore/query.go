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

func (s *StoreProtocol) getHeaders(param *types.ReqBlocks) *types.Headers {
	for _, pid := range s.Discovery.RoutingTale() {
		headers, err := s.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err)
			continue
		}
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return nil
}

func (s *StoreProtocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, GetHeader)
	if err != nil {
		return nil, err
	}
	msg := types2.Message{
		ProtocolID: GetHeader,
		Params:     param,
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	b, _ := json.Marshal(msg)
	_, err = rw.Write(b)
	if err != nil {
		log.Error("getHeadersFromPeer", "stream write error", err)
		return nil, err
	}
	rw.Flush()
	stream.Close()
	//close之后不能写数据，但依然可以读数据
	res, err := readResponse(stream)
	if err != nil {
		return nil, err
	}
	headers, ok := res.Result.(*types.Headers)
	if !ok {
		return nil, types2.ErrInvalidResponse
	}
	return headers, nil
}

func (s *StoreProtocol) getChunkRecords(param *types.ReqChunkRecords) *types.ChunkRecords {
	for _, pid := range s.Discovery.RoutingTale() {
		records, err := s.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err)
			continue
		}
		return records
	}

	log.Error("getChunkRecords", "error", types2.ErrNotFound)
	return nil
}

func (s *StoreProtocol) getChunkRecordsFromPeer(param *types.ReqChunkRecords, pid peer.ID) (*types.ChunkRecords, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, GetChunkRecord)
	if err != nil {
		return nil, err
	}
	msg := types2.Message{
		ProtocolID: GetChunkRecord,
		Params:     param,
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	b, _ := json.Marshal(msg)
	_, err = rw.Write(b)
	if err != nil {
		log.Error("getChunkRecordsFromPeer", "stream write error", err)
		return nil, err
	}
	rw.Flush()
	//close之后不能写数据，但依然可以读数据
	stream.Close()
	res, err := readResponse(stream)
	if err != nil {
		return nil, err
	}
	records, ok := res.Result.(*types.ChunkRecords)
	if !ok {
		return nil, types2.ErrInvalidResponse
	}
	return records, nil
}

// fetchChunkOrNearerPeersAsync 返回 *types.ChunkBlockBody 或者 []peer.ID
func (s *StoreProtocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ReqChunkBlockBody, peers []peer.ID) (*types.BlockBodys, []peer.ID) {

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
	for range peers {
		res := <-responseCh
		if res == nil || res.Error != nil {
			continue
		}
		switch t := res.Result.(type) {
		case *types.BlockBodys:
			//查到了区块数据，直接返回
			return t, nil
		case []peer.AddrInfo:
			//没查到区块数据，返回了更近的节点信息
			for _, addrInfo := range t {
				peerList = append(peerList, addrInfo.ID)
			}

		default:
			//返回类型不是*types.BlockBodys或[]peer.ID，对端节点异常
			log.Error("fetchChunkAsync", "fetchChunkOrNearerPeers invalid response", res.Result)

		}
	}

	//TODO 若超过3个，排序选择最优的三个节点
	if len(peerList) > AlphaValue {
		peerList = peerList[:AlphaValue]
	}

	return nil, peerList
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
	_, err = rw.Write(b)
	if err != nil {
		log.Error("fetchChunkOrNearerPeers", "stream write error", err)
		return nil
	}
	rw.Flush()
	res, err := readResponse(stream)
	if err != nil {
		log.Error("fetchChunkFromPeer", "read response error", err)
		return nil
	}
	log.Info("fetchChunkOrNearerPeers response ok", "remote peer", stream.Conn().RemotePeer().Pretty())

	if addrInfos, ok := res.Result.([]peer.AddrInfo); ok {
		for _, addrInfo := range addrInfos {
			s.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
		}
	}
	//res.Result可能是*types.BlockBodys或者[]peer.AddrInfo
	return res
}

func (s *StoreProtocol) getChunkFromBlockchain(param *types.ChunkInfo) (*types.BlockBodys, error) {
	msg := s.QueueClient.NewMessage("blockchain", types.EventGetChunkBlockBody, param)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.BlockBodys), nil
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
