package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (s *StoreProtocol) getHeaders(param *types.ReqBlocks) *types.Headers {
	for _, pid := range s.Discovery.RoutingTale() {
		headers, err := s.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return &types.Headers{}
}

func (s *StoreProtocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, GetHeader)
	if err != nil {
		return nil, err
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	fmt.Println("getHeadersFromPeer", "local", stream.Conn().LocalPeer(), "remote", stream.Conn().RemotePeer(), "id", s.Host.ID())
	msg := types.P2PStoreRequest{
		ProtocolID: GetHeader,
		Data: &types.P2PStoreRequest_ReqBlocks{
			ReqBlocks: param,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("getHeadersFromPeer", "stream write error", err)
		return nil, err
	}
	stream.Close()
	//close之后不能写数据，但依然可以读数据
	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		return nil, err
	}
	return res.Result.(*types.P2PStoreResponse_Headers).Headers, nil
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
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: GetChunkRecord,
		Data: &types.P2PStoreRequest_ReqChunkRecords{
			ReqChunkRecords: param,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("getChunkRecordsFromPeer", "stream write error", err)
		return nil, err
	}
	//close之后不能写数据，但依然可以读数据
	stream.Close()
	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		return nil, err
	}
	return res.Result.(*types.P2PStoreResponse_ChunkRecords).ChunkRecords, nil
}

// fetchChunkOrNearerPeersAsync 返回 *types.ChunkBlockBody 或者 []peer.ID
func (s *StoreProtocol) fetchChunkOrNearerPeersAsync(ctx context.Context, param *types.ReqChunkBlockBody, peers []peer.ID) (*types.BlockBodys, []peer.ID) {

	responseCh := make(chan interface{}, AlphaValue)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	// *3* 个节点并发请求
	for _, peerID := range peers {
		if peerID == s.Host.ID() {
			responseCh <- nil
			continue
		}
		go func(pid peer.ID) {
			bodys, addrInfos, err := s.fetchChunkOrNearerPeers(cancelCtx, param, pid)
			if err != nil {
				log.Error("fetchChunkOrNearerPeersAsync", "fetchChunkOrNearerPeers error", err, "peer id", pid)
				responseCh <- nil
			}
			if bodys != nil {
				responseCh <- bodys
			} else if len(addrInfos) != 0 {
				responseCh <- addrInfos
			}
		}(peerID)
	}

	var peerList []peer.ID
	for range peers {
		res := <-responseCh
		switch t := res.(type) {
		case *types.BlockBodys:
			//查到了区块数据，直接返回
			return t, nil
		case []peer.AddrInfo:
			//没查到区块数据，返回了更近的节点信息
			for _, addrInfo := range t {
				peerList = append(peerList, addrInfo.ID)
			}
		}
	}

	//TODO 若超过3个，排序选择最优的三个节点
	if len(peerList) > AlphaValue {
		peerList = peerList[:AlphaValue]
	}

	return nil, peerList
}

func (s *StoreProtocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ReqChunkBlockBody, pid peer.ID) (*types.BlockBodys, []peer.AddrInfo, error) {
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stream, err := s.Host.NewStream(childCtx, pid, FetchChunk)
	if err != nil {
		log.Error("getBlocksFromRemote", "error", err)
		return nil, nil, err
	}
	//defer stream.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: FetchChunk,
		Data: &types.P2PStoreRequest_ReqChunkBlockBody{
			ReqChunkBlockBody: params,
		},
	}
	err = writeMessage(rw.Writer, &msg)
	if err != nil {
		log.Error("fetchChunkOrNearerPeers", "stream write error", err)
		return nil, nil, err
	}
	var res types.P2PStoreResponse
	err = readMessage(rw.Reader, &res)
	if err != nil {
		log.Error("fetchChunkFromPeer", "read response error", err, "multiaddr", stream.Conn().LocalMultiaddr())
		return nil, nil, err
	}
	log.Info("fetchChunkOrNearerPeers response ok", "remote peer", stream.Conn().RemotePeer().Pretty())

	switch v := res.Result.(type) {
	case *types.P2PStoreResponse_BlockBodys:
		if params.Filter {
			var bodyList []*types.BlockBody
			for _, body := range v.BlockBodys.Items {
				if body.Height >= params.Start && body.Height <= params.End {
					bodyList = append(bodyList, body)
				}
			}
			v.BlockBodys.Items = bodyList
		}
		return v.BlockBodys, nil, nil
	case *types.P2PStoreResponse_AddrInfo:
		var addrInfos []peer.AddrInfo
		err = json.Unmarshal(v.AddrInfo, &addrInfos)
		if err != nil {
			log.Error("fetchChunkOrNearerPeers", "addrInfo error", err)
		}
		//如果对端节点返回了addrInfo，把节点信息加入到PeerStore
		for _, addrInfo := range addrInfos {
			s.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
		}
		return nil, addrInfos, nil
	default:
		if res.ErrorInfo != "" {
			return nil, nil, errors.New(res.ErrorInfo)
		}
	}

	return nil, nil, types2.ErrNotFound
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
