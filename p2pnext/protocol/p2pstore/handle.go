package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"time"

	protocol2 "github.com/33cn/chain33/p2pnext/protocol"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/gogo/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

const (
	FetchChunk     = "/chain33/fetch-chunk/1.0.0"
	StoreChunk     = "/chain33/store-chunk/1.0.0"
	GetHeader      = "/chain33/headers/1.0.0"
	GetChunkRecord = "/chain33/chunk-record/1.0.0"
)

type StoreProtocol struct {
	protocol2.BaseProtocol //default协议实现
	*protocol2.P2PEnv      //协议共享接口变量
}

func Init(env *protocol2.P2PEnv) {
	p := &StoreProtocol{
		P2PEnv: env,
	}

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(StoreChunk, p.Handle)
	p.Host.SetStreamHandler(FetchChunk, p.Handle)
	p.Host.SetStreamHandler(GetHeader, p.Handle)
	p.Host.SetStreamHandler(GetChunkRecord, p.Handle)
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol2.RegisterEventHandler(types.EventNotifyStoreChunk, p.HandleEvent)
	protocol2.RegisterEventHandler(types.EventGetChunkBlock, p.HandleEvent)
	protocol2.RegisterEventHandler(types.EventGetChunkBlockBody, p.HandleEvent)
	protocol2.RegisterEventHandler(types.EventGetChunkRecord, p.HandleEvent)
}

// Handle 处理节点之间的请求
func (s *StoreProtocol) Handle(stream core.Stream) {

	msg, err := readMessage(stream)
	if err != nil {
		log.Error("handle", "unmarshal error", err)
		_ = stream.Reset()
		return
	}

	//具体业务处理逻辑
	switch msg.ProtocolID {
	//不同的协议交给不同的处理逻辑
	case FetchChunk:
		s.onFetchChunk(stream, msg.Params)
	case StoreChunk:
		s.onStoreChunk(stream, msg.Params)
	case GetHeader:
		s.onGetHeader(stream, msg.Params)
	case GetChunkRecord:
		s.onGetChunkRecord(stream, msg.Params)
	default:
		log.Error("Handle", "error", types2.ErrProtocolNotSupport, "protocol", msg.ProtocolID)
	}
	stream.Close()
}

// HandleEvent 处理模块之间的事件
func (s *StoreProtocol) HandleEvent(m *queue.Message) {
	switch m.Ty {

	// 通知临近节点进行区块数据归档
	case types.EventNotifyStoreChunk:
		data := m.GetData().(*types.ChunkInfo)
		err := s.StoreChunk(data)
		if err != nil {
			log.Error("HandleEvent", "storeChunk error", err)
			return
		}

	// 获取chunkBlock数据
	case types.EventGetChunkBlock:
		req := m.GetData().(*types.ReqChunkBlock)
		bodys, err := s.GetChunk(&types.ReqChunkBlockBody{
			ChunkHash: req.ChunkHash,
			Filter:    true,
			Start:     req.Start,
			End:       req.End,
		})
		if err != nil {
			log.Error("HandleEvent", "GetChunk error", err)
			return
		}
		headers := s.getHeaders(&types.ReqBlocks{Start: req.Start, End: req.End})
		if len(headers.Items) != len(bodys.Items) {
			log.Error("GetBlockHeader", "error", types2.ErrLength, "header length", len(headers.Items), "body length", len(bodys.Items))
			return
		}

		var blockList []*types.Block
		for index := range bodys.Items {
			body := bodys.Items[index]
			header := headers.Items[index]
			block := &types.Block{
				Version:    header.Version,
				ParentHash: header.ParentHash,
				TxHash:     header.TxHash,
				StateHash:  header.StateHash,
				Height:     header.Height,
				BlockTime:  header.BlockTime,
				Difficulty: header.Difficulty,
				MainHash:   body.MainHash,
				MainHeight: body.MainHeight,
				Signature:  header.Signature,
				Txs:        body.Txs,
			}
			blockList = append(blockList, block)
		}
		msg := s.QueueClient.NewMessage("blockchain", types.EventReplyChunkBlock, &types.Blocks{Items: blockList})
		err = s.QueueClient.Send(msg, false)
		if err != nil {
			log.Error("EventGetChunkBlock", "reply message error", err)
		}

	// 获取chunkBody数据
	case types.EventGetChunkBlockBody:
		req := m.GetData().(*types.ReqChunkBlockBody)
		blockBodys, err := s.GetChunk(req)
		if err != nil {
			log.Error("HandleEvent", "GetChunkBlockBody error", err)
			return
		}
		msg := s.QueueClient.NewMessage("blockchain", types.EventReplyChunkBlockBody, blockBodys)
		err = s.QueueClient.Send(msg, false)
		if err != nil {
			log.Error("EventGetChunkBlockBody", "reply message error", err)
		}

	// 获取归档索引
	case types.EventGetChunkRecord:
		req := m.GetData().(*types.ReqChunkRecords)
		records := s.getChunkRecords(req)
		if records == nil {
			log.Error("HandleEvent", "GetChunkRecord error", types2.ErrNotFound)
			return
		}
		msg := s.QueueClient.NewMessage("blockchain", types.EventReplyChunkRecord, records)
		err := s.QueueClient.Send(msg, false)
		if err != nil {
			log.Error("EventGetChunkBlockBody", "reply message error", err)
		}
	}
}

// VerifyRequest  验证请求数据
func (s *StoreProtocol) VerifyRequest(message proto.Message, messageComm *types.MessageComm) bool {
	return true
}

// SignMessage 对消息签名
func (s *StoreProtocol) SignProtoMessage(message proto.Message) ([]byte, error) {
	return nil, nil
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
	if ok, _ := s.DB.Has(genChunkKey(req.ChunkHash)); ok {
		//本地有数据
		b, err := s.DB.Get(genChunkKey(req.ChunkHash))
		if err != nil {
			res.Error = err
			return
		}
		var data types2.StorageData
		err = json.Unmarshal(b, &data)
		if err != nil {
			res.Error = err
			return
		}
		//本地有数据且没有过期
		if time.Since(data.RefreshTime) < types2.ExpiredTime {
			var blocks *types.BlockBodys
			_ = json.Unmarshal(data.Data.([]byte), &blocks) //请求方可以要求过滤出指定的区块高度，若不要求过滤则返回该归档数据包含的所有区块
			if req.Filter {
				var bodyList []*types.BlockBody
				for _, body := range blocks.Items {
					if body.Height >= req.Start && body.Height <= req.End {
						bodyList = append(bodyList, body)
					}
				}
				blocks.Items = bodyList
			}
			res.Result = blocks
			return
		}
		//本地有数据但过期了
		err = s.deleteChunkBlock(req.ChunkHash)
		if err != nil {
			log.Error("onFetchChunk", "delete chunk error", err)
		}
	}

	//本地没有数据或本地数据已过期
	peers := s.Discovery.Routing().RoutingTable().NearestPeers(kbt.ConvertPeerID(s.Host.ID()), AlphaValue)
	res.Result = peers

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
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res types2.Response
	defer func() {
		b, _ := json.Marshal(res)
		_, err := rw.Write(b)
		if err != nil {
			log.Error("onStoreChunk", "stream write error", err)
		}
		rw.Flush()
	}()
	req, ok := in.(*types.ChunkInfo)
	if !ok {
		res.Error = types2.ErrInvalidParam
		return
	}

	var b []byte
	var err error
	b, err = s.DB.Get(genChunkKey(req.ChunkHash))
	if err == nil {
		//本节点p2pStore已有数据
		var data types2.StorageData
		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Error("onStoreChunk", "unmarshal error", err)
			res.Error = types2.ErrUnexpected
			return
		}
		data.RefreshTime = time.Now()
		b, _ = json.Marshal(data)
		err = s.DB.Put(genChunkKey(req.ChunkHash), b)
		if err != nil {
			log.Error("onStoreChunk", "unmarshal error", err)
			res.Error = types2.ErrDBSave
		}
		return
	}

	var bodys *types.BlockBodys
	bodys, err = s.getChunkFromBlockchain(in)
	if err != nil {
		//本地节点没有数据，则从对端节点请求数据
		req := in.(*types.ChunkInfo)
		res2 := s.fetchChunkOrNearerPeers(context.Background(), &types.ReqChunkBlockBody{ChunkHash: req.ChunkHash}, stream.Conn().RemotePeer())
		//对端节点发过来的消息，对端节点一定有数据
		if res2 == nil {
			res.Error = types2.ErrNotFound
			return
		}
		if res2.Error != nil {
			res.Error = res2.Error
			return
		}
		bodys = res2.Result.(*types.BlockBodys)

	}

	b, err = json.Marshal(types2.StorageData{
		Data:        bodys,
		RefreshTime: time.Now(),
	})
	if err != nil {
		res.Error = err
		return
	}

	hash := req.ChunkHash
	err = s.DB.Put(genChunkKey(hash), b)
	if err != nil {
		res.Error = err
		return
	}
	err = s.addLocalChunkInfo(req)
	if err != nil {
		//索引存储失败，存储的区块也要回滚
		_ = s.DB.Delete(genChunkKey(hash))
		res.Error = err
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

func (s *StoreProtocol) getChunkFromBlockchain(param interface{}) (*types.BlockBodys, error) {
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
