package p2pstore

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"time"

	protocol2 "github.com/33cn/chain33/p2pnext/protocol"
	types2 "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	ggio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-datastore"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

const (
	FetchData = "/chain33/fetch-data/1.0.0"
	PutData   = "/chain33/put-data/1.0.0"
)

//var protocolIDs []protocol.ID
//
//func init() {
//	protocolIDs = append(protocolIDs, FetchData)
//	protocolIDs = append(protocolIDs, PutData)
//}

type StoreProtocol struct {
	protocol2.BaseProtocol //default协议实现
	protocol2.Global       //获取全局变量接口
}

func New(global protocol2.Global) protocol2.Protocol {
	p := &StoreProtocol{
		Global: global,
	}
	//实例化之后同时注册eventHandler
	//TODO evnetID修改为真实的
	protocol2.RegisterEventHandler(0, p.HandleEvent)
	return p
}

// Handle 处理节点之间的请求
func (s *StoreProtocol) Handle(stream core.Stream) {
	var data []byte
	var err error
	defer func() {
		if err != nil {
			stream.Reset()
		} else {
			stream.Close()
		}
	}()
	for {
		buf := make([]byte, 100)
		var n int
		n, err = stream.Read(buf)
		if err != nil && err != io.EOF {
			//TODO +log
			return
		}
		data = append(data, buf[:n]...)
		if err == io.EOF {
			break
		}
	}

	var msg protocol2.Message
	err = json.Unmarshal(data, &msg)
	if err != nil {
		//TODO +log
		return
	}

	//具体业务处理逻辑
	switch msg.ProtocolID {
	//不同的协议交给不同的处理逻辑
	case FetchData:
		s.onFetchData(stream, msg.Params)
	case PutData:
		s.onPutData(stream, msg.Params)
	default:
	}
}

// HandleEvent 处理模块之间的事件
func (s *StoreProtocol) HandleEvent(m *queue.Message) {

}

// Request 向指定节点发送请求
// TODO 放到host目录下
func (s *StoreProtocol) Request(id peer.ID, p protocol.ID, data proto.Message) error {
	stream, err := s.Host().NewStream(context.Background(), id, p)
	if err != nil {
		return err
	}
	writer := ggio.NewFullWriter(stream)
	err = writer.WriteMsg(data)
	if err != nil {
		_ = stream.Reset()
		return err
	}

	err = helpers.FullClose(stream)
	if err != nil {
		_ = stream.Reset()
		return err
	}
	return nil
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
func (s *StoreProtocol) onFetchData(stream core.Stream, in interface{}) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res protocol2.Response
	defer func() {
		b, _ := json.Marshal(res)
		rw.Write(b)
		rw.Flush()
	}()
	data, ok := in.(*types2.FetchBlocks)
	if !ok {
		res.Error = types2.ErrParams
		return
	}

	if ok, _ := s.DB().Has(packageKey(data.Hash)); ok {
		b, err := s.DB().Get(packageKey(data.Hash))
		if err != nil {
			res.Error = err
			return
		}
		var blocks []*types.Block
		err = json.Unmarshal(b, &blocks)
		if err != nil {
			res.Error = err
			return
		}
		res.Result = blocks
		//请求方可以要求过滤出指定的区块高度，若不要求过滤则返回该归档数据包含的所有区块
		if data.Filter {
			var filterBlocks []*types.Block
			for _, block := range blocks {
				if block.Height >= data.Start && block.Height <= data.End {
					filterBlocks = append(filterBlocks, block)
				}
			}
			res.Result = filterBlocks
		}
		return
	}
	peers := s.Discovery().KademliaDHT.RoutingTable().NearestPeers(kbt.ConvertPeerID(s.Host().ID()), AlphaValue)
	res.Result = peers
	return

}

func (s *StoreProtocol) onPutData(stream core.Stream, in interface{}) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	var res protocol2.Response
	defer func() {
		b, _ := json.Marshal(res)
		rw.Write(b)
		rw.Flush()
	}()
	data, ok := in.(*types2.PutPackage)
	if !ok {
		res.Error = types2.ErrParams
		return
	}
	if packageHash(data.Blocks) != data.Hash {
		res.Error = ErrCheckSum
		return
	}

	b, err := json.Marshal(data.Blocks)
	if err != nil {
		res.Error = err
		return
	}
	err = s.DB().Put(datastore.NewKey(BlocksPrefix+data.Hash), b)
	if err != nil {
		res.Error = err
		return
	}
	err = s.AddLocalIndexHash(data.Hash)
	if err != nil {
		//索引存储失败，存储的区块也要回滚
		_ = s.DB().Delete(datastore.NewKey(BlocksPrefix + data.Hash))
		res.Error = err
		return
	}
}

func (s *StoreProtocol) getBlocksFromRemote(ctx context.Context, params *types2.FetchBlocks, pid peer.ID, ch chan<- protocol2.Response) {
	childCtx, _ := context.WithTimeout(ctx, 3*time.Minute)
	stream, err := s.Host().NewStream(childCtx, pid, FetchData)
	if err != nil {
		//TODO +log
		return
	}
	defer stream.Close()
	msg := protocol2.Message{
		ProtocolID: FetchData,
		Params:     params,
	}
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	b, _ := json.Marshal(msg)
	rw.Write(b)
	rw.Flush()
	res, _ := readResponse(stream)
	ch <- res
}
func readResponse(stream network.Stream) (protocol2.Response, error) {
	var data []byte
	var err error
	for {
		buf := make([]byte, 100)
		var n int
		n, err = stream.Read(buf)
		if err != nil && err != io.EOF {
			return protocol2.Response{}, err
		}
		data = append(data, buf[:n]...)
		if err == io.EOF {
			break
		}
	}

	var res protocol2.Response
	err = json.Unmarshal(data, &res)
	if err != nil {
		return protocol2.Response{}, err
	}

	return res, nil
}
