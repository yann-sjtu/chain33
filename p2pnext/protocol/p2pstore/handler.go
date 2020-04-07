package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"time"

	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

//StoreChunk handles notification of blockchain,
// store chunk if this node is the nearest node in the local routing table.
func (s *StoreProtocol) StoreChunk(req *types.ChunkInfo) error {
	if req == nil || req.Start > req.End || req.End-req.Start > types2.MaxChunkNum {
		return types2.ErrInvalidParam
	}

	//路由表中存在比当前节点更近的节点，说明当前节点不是局部最优节点，不需要保存数据
	pid := s.Discovery.Routing().RoutingTable().NearestPeer(kb.ConvertKey(genChunkPath(req.ChunkHash)))
	if pid != "" && kb.Closer(pid, s.Host.ID(), genChunkPath(req.ChunkHash)) {
		return nil
	}
	log.Info("StoreChunk", "local pid", s.Host.ID(), "chunk num", req.ChunkNum)
	//如果p2pStore已保存数据，只更新时间即可
	if err := s.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := s.getChunkFromBlockchain(req)
	if err != nil {
		log.Error("StoreChunk", "getChunkFromBlockchain error", err)
		return err
	}
	err = s.addChunkBlock(req, bodys)
	if err != nil {
		log.Error("StoreChunk", "addChunkBlock error", err)
		return err
	}

	//本地存储之后立即到其他节点做一次备份
	s.notifyStoreChunk(req)
	return nil
}

//GetChunk gets chunk data from p2pStore or other peers.
func (s *StoreProtocol) GetChunk(req *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地p2pStore数据
	bodys, err := s.getChunkBlock(req.ChunkHash)
	if err == nil {
		if req.Filter {
			var bodyList []*types.BlockBody
			for _, body := range bodys.Items {
				if body.Height >= req.Start && body.Height <= req.End {
					bodyList = append(bodyList, body)
				}
			}
			bodys.Items = bodyList
		}
		return bodys, nil
	}

	//本地数据不存在或已过期，则向临近节点查询
	//首先从本地路由表获取 *3* 个最近的节点
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kb.ConvertKey(genChunkPath(req.ChunkHash)), AlphaValue)
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

// 其他节点向本节点请求数据时，本地存在则直接返回，不存在则返回更近的多个节点
func (s *StoreProtocol) onFetchChunk(writer *bufio.Writer, req *types.ReqChunkBlockBody) {
	var res types.P2PStoreResponse
	defer func() {
		err := writeMessage(writer, &res)
		if err != nil {
			log.Error("onFetchChunk", "stream write error", err)
		}
	}()
	//优先检查本地是否存在
	bodys, err := s.getChunkBlock(req.ChunkHash)
	if err == nil {
		if req.Filter {
			var bodyList []*types.BlockBody
			for _, body := range bodys.Items {
				if body.Height >= req.Start && body.Height <= req.End {
					bodyList = append(bodyList, body)
				}
			}
			bodys.Items = bodyList
		}
		res.Result = &types.P2PStoreResponse_BlockBodys{BlockBodys: bodys}
		return
	}

	//本地没有数据或本地数据已过期
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kb.ConvertPeerID(s.Host.ID()), AlphaValue)
	var addrInfos []peer.AddrInfo
	for _, pid := range peers {
		addrInfos = append(addrInfos, peer.AddrInfo{
			ID:    pid,
			Addrs: s.Host.Peerstore().Addrs(pid),
		})
	}
	addrInfosData, err := json.Marshal(addrInfos)
	if err != nil {
		log.Error("onFetchChunk", "addr info marshal error", err)
	}
	res.Result = &types.P2PStoreResponse_AddrInfo{AddrInfo: addrInfosData}

}

// 对端节点通知本节点保存数据
/*
检查本节点p2pStore是否保存了数据，
	1）若已保存则只更新时间即可
	2）若未保存：
		1. 向blockchain模块请求
		2. blockchain模块没有数据则向对端节点请求
*/
func (s *StoreProtocol) onStoreChunk(stream network.Stream, req *types.ChunkInfo) {
	//检查本地 p2pStore，如果已存在数据则直接更新
	err := s.updateChunk(req)
	if err == nil {
		return
	}

	//本地 p2pStore没有数据，向blockchain请求数据
	bodys, err := s.getChunkFromBlockchain(req)
	if err != nil {
		//本地节点没有数据，则从对端节点请求数据
		s.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), stream.Conn().RemoteMultiaddr(), time.Hour)
		bodys, _, err = s.fetchChunkOrNearerPeers(context.Background(), &types.ReqChunkBlockBody{ChunkHash: req.ChunkHash}, stream.Conn().RemotePeer())
		//对端节点发过来的消息，对端节点一定有数据
		if err != nil {
			log.Error("onStoreChunk", "get bodys from remote peer error", err)
			return
		}
	}

	err = s.addChunkBlock(req, bodys)
	if err != nil {
		log.Error("onStoreChunk", "store block error", err)
		return
	}
}

func (s *StoreProtocol) onGetHeader(writer *bufio.Writer, req *types.ReqBlocks) {
	var res types.P2PStoreResponse
	defer func() {
		err := writeMessage(writer, &res)
		if err != nil {
			log.Error("onGetHeader", "stream write error", err)
		}
	}()

	msg := s.QueueClient.NewMessage("blockchain", types.EventGetHeaders, req)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		res.ErrorInfo = err.Error()
		return
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		res.ErrorInfo = err.Error()
		return
	}

	if reply, ok := resp.GetData().(*types.Reply); ok {
		res.ErrorInfo = string(reply.Msg)
		return
	}
	res.Result = &types.P2PStoreResponse_Headers{Headers: resp.GetData().(*types.Headers)}
}

func (s *StoreProtocol) onGetChunkRecord(writer *bufio.Writer, req *types.ReqChunkRecords) {
	var res types.P2PStoreResponse
	defer func() {
		err := writeMessage(writer, &res)
		if err != nil {
			log.Error("onGetChunkRecord", "stream write error", err)
		}
	}()

	msg := s.QueueClient.NewMessage("blockchain", types.EventGetChunkRecord, req)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		res.ErrorInfo = err.Error()
		return
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		res.ErrorInfo = err.Error()
		return
	}
	if reply, ok := resp.GetData().(*types.Reply); ok {
		res.ErrorInfo = string(reply.Msg)
		return
	}
	res.Result = &types.P2PStoreResponse_ChunkRecords{ChunkRecords: resp.GetData().(*types.ChunkRecords)}
}

func readMessage(reader *bufio.Reader, msg types.Message) error {
	var data []byte
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		data = append(data, buf[:n]...)
		if n < 1024 {
			break
		}
	}
	return types.Decode(data, msg)
}

func writeMessage(writer *bufio.Writer, msg types.Message) error {
	b := types.Encode(msg)
	_, err := writer.Write(b)
	if err != nil {
		return err
	}
	return writer.Flush()
}
