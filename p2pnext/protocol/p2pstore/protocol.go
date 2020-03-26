package p2pstore

import (
	"github.com/33cn/chain33/p2pnext/protocol"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"

	"github.com/gogo/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
)

const (
	FetchChunk     = "/chain33/fetch-chunk/1.0.0"
	StoreChunk     = "/chain33/store-chunk/1.0.0"
	GetHeader      = "/chain33/headers/1.0.0"
	GetChunkRecord = "/chain33/chunk-record/1.0.0"
)

type StoreProtocol struct {
	protocol.BaseProtocol //default协议实现
	*protocol.P2PEnv      //协议共享接口变量
}

func Init(env *protocol.P2PEnv) {
	p := &StoreProtocol{
		P2PEnv: env,
	}

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(StoreChunk, p.Handle)
	p.Host.SetStreamHandler(FetchChunk, p.Handle)
	p.Host.SetStreamHandler(GetHeader, p.Handle)
	p.Host.SetStreamHandler(GetChunkRecord, p.Handle)
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkBlock, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, p.HandleEvent)
	protocol.RegisterEventHandler(types.EventGetChunkRecord, p.HandleEvent)
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
		msg := s.QueueClient.NewMessage("blockchain", types.EventAddChunkBlock, &types.Blocks{Items: blockList})
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
		m.Reply(&queue.Message{Data: blockBodys})

	// 获取归档索引
	case types.EventGetChunkRecord:
		req := m.GetData().(*types.ReqChunkRecords)
		records := s.getChunkRecords(req)
		if records == nil {
			log.Error("HandleEvent", "GetChunkRecord error", types2.ErrNotFound)
			return
		}
		msg := s.QueueClient.NewMessage("blockchain", types.EventAddChunkRecord, records)
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
