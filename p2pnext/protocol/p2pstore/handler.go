package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"time"

	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

//StoreChunk handles notification of blockchain
// store chunk if this node is the nearest node in the local routing table
func (s *StoreProtocol) StoreChunk(req *types.ChunkInfo) error {
	if req == nil {
		return types2.ErrInvalidParam
	}

	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kb.ConvertKey(genChunkPath(req.ChunkHash)), 1)
	if len(peers) != 0 && kb.Closer(peers[0], s.Host.ID(), genChunkPath(req.ChunkHash)) {
		return nil
	}
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

	//本地存储之后立即到其他节点做依次备份
	s.notifyStoreChunk(req)
	return nil
}

func (s *StoreProtocol) GetChunk(req *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地数据
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
func (s *StoreProtocol) onFetchChunk(stream core.Stream, in interface{}) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res types2.Response
	defer func() {
		b, _ := json.Marshal(res)
		_, err := rw.Write(b)
		if err != nil {
			log.Error("onFetchChunk", "stream write error", err)
		}
		rw.Flush()
	}()

	req, ok := in.(*types.ReqChunkBlockBody)
	if !ok {
		res.Error = types2.ErrInvalidParam
		return
	}
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
		res.Result = bodys
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
	res.Result = addrInfos

}

// 对端节点通知本节点保存数据
/*
检查本节点p2pStore是否保存了数据，
	1）若已保存则只更新时间即可
	2）若未保存：
		1. 向blockchain模块请求
		2. blockchain模块没有数据则向对端节点请求
*/
func (s *StoreProtocol) onStoreChunk(stream core.Stream, in interface{}) {
	defer stream.Conn().Close()

	req, ok := in.(*types.ChunkInfo)
	if !ok {
		log.Error("onStoreChunk", "invalid param", in)
		return
	}

	//检查本地 p2pStore
	bodys, err := s.getChunkBlock(req.ChunkHash)
	if err == nil {
		err = s.addChunkBlock(req, bodys)
		if err != nil {
			log.Error("onStoreChunk", "update local chunk error", err)
		}
		return
	}

	//本地 p2pStore没有数据，向blockchain请求数据
	bodys, err = s.getChunkFromBlockchain(req)
	if err != nil {
		//本地节点没有数据，则从对端节点请求数据
		s.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), stream.Conn().RemoteMultiaddr(), time.Hour)
		res2 := s.fetchChunkOrNearerPeers(context.Background(), &types.ReqChunkBlockBody{ChunkHash: req.ChunkHash}, stream.Conn().RemotePeer())
		//对端节点发过来的消息，对端节点一定有数据
		if res2 == nil || res2.Error != nil {
			log.Error("onStoreChunk", "get bodys from remote peer error", "invalid response", "response", res2)
			return
		}
		var ok bool
		if bodys, ok = res2.Result.(*types.BlockBodys); !ok {
			log.Error("onStoreChunk", "get bodys from remote peer error", "invalid response", "result", res2.Result)
			return
		}

	}

	err = s.addChunkBlock(req, bodys)
	if err != nil {
		log.Error("onStoreChunk", "store block error", err)
		return
	}
}

func (s *StoreProtocol) onGetHeader(stream core.Stream, in interface{}) {
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res types2.Response
	defer func() {
		b, _ := json.Marshal(res)
		_, err := rw.Write(b)
		if err != nil {
			log.Error("onGetHeader", "stream write error", err)
		}
		rw.Flush()
	}()
	req, ok := in.(*types.ReqBlocks)
	if !ok {
		res.Error = types2.ErrInvalidParam
		return
	}
	msg := s.QueueClient.NewMessage("blockchain", types.EventHeaders, req)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		res.Error = err
		return
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		res.Error = err
		return
	}
	res.Result = resp.GetData()
}

func (s *StoreProtocol) onGetChunkRecord(stream core.Stream, in interface{}) {
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res types2.Response
	defer func() {
		b, _ := json.Marshal(res)
		_, err := rw.Write(b)
		if err != nil {
			log.Error("onGetChunkRecord", "stream write error", err)
		}
		rw.Flush()
	}()
	req, ok := in.(*types.ReqChunkRecords)
	if !ok {
		res.Error = types2.ErrInvalidParam
		return
	}

	msg := s.QueueClient.NewMessage("blockchain", types.EventGetChunkRecord, req)
	err := s.QueueClient.Send(msg, true)
	if err != nil {
		res.Error = err
		return
	}
	resp, err := s.QueueClient.Wait(msg)
	if err != nil {
		res.Error = err
		return
	}
	res.Result = resp.GetData()
}
