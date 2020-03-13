package protocol

import (
	manage2 "github.com/33cn/chain33/p2p/manage"
	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/gogo/protobuf/proto"
	ds "github.com/ipfs/go-datastore"
	core "github.com/libp2p/go-libp2p-core"
)

// Protocol 所有协议实现都必须实现的接口
type Protocol interface {

	// VerifyRequest  验证请求数据
	VerifyRequest(message proto.Message, messageComm *types.MessageComm) bool
	// SignMessage 对消息签名
	SignProtoMessage(message proto.Message) ([]byte, error)
	// Handle 处理节点之间的请求
	Handle(stream core.Stream)
	// HandleEvent 处理模块之间的事件
	HandleEvent(m *queue.Message)
}

// BaseProtocol BaseProtocol是协议方法的公共实现，所有协议实现都可以继承该接口
type BaseProtocol struct{}

// VerifyRequest  验证请求数据
func (b BaseProtocol) VerifyRequest(message proto.Message, messageComm *types.MessageComm) bool {
	return true
}

// SignMessage 对消息签名
func (b BaseProtocol) SignProtoMessage(message proto.Message) ([]byte, error) {
	return nil, nil
}

// Handle 处理节点之间的请求
func (b BaseProtocol) Handle(stream core.Stream) {}

// HandleEvent 处理模块之间的事件
func (b BaseProtocol) HandleEvent(m *queue.Message) {}

//Message 协议传递时统一消息类型
// Protocol指定具体协议
// Data是消息体
type Message struct {
	ProtocolID string
	Params     interface{}
}

type Response struct {
	Result interface{}
	Error  error
}

// P2PEnv p2p全局公共变量
type P2PEnv struct {
	ChainCfg        *types.Chain33Config
	QueueClient     queue.Client
	Host            core.Host
	ConnManager     *manage.ConnManager
	PeerInfoManager *manage.PeerInfoManager
	Discovery       *dht.Discovery
	P2PManager      *manage2.P2PMgr
	SubConfig       *p2pty.P2PSubConfig
	DB              ds.Datastore
}
