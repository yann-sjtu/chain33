package p2pnext

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/client"
	logger "github.com/33cn/chain33/common/log/log15"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
	"github.com/33cn/chain33/p2pnext/addrbook"
	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"
	"github.com/33cn/chain33/p2pnext/protocol"
	"github.com/33cn/chain33/p2pnext/protocol/p2pstore"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/p2pnext/store"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/multiformats/go-multiaddr"
)

var log = logger.New("module", "p2pnext")

func init() {
	p2pmgr.RegisterP2PCreate(p2pmgr.DHTTypeName, New)
}

type P2P struct {
	chainCfg        *types.Chain33Config
	host            core.Host
	discovery       *dht.Discovery
	connManager     *manage.ConnManager
	peerInfoManager *manage.PeerInfoManager
	api             client.QueueProtocolAPI
	client          queue.Client
	addrBook        *addrbook.AddrBook
	taskGroup       *sync.WaitGroup
	closed          int32
	p2pCfg          *types.P2P
	subCfg          *p2pty.P2PSubConfig
	mgr             *p2pmgr.P2PMgr
	subChan         chan interface{}
	db              ds.Datastore

	env *protocol.P2PEnv
}

func New(mgr *p2pmgr.P2PMgr, subCfg []byte) p2pmgr.IP2P {

	chainCfg := mgr.ChainCfg
	p2pCfg := chainCfg.GetModuleConfig().P2P
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(subCfg, mcfg)
	if mcfg.Port == 0 {
		mcfg.Port = 13803
	}

	addrbook := addrbook.NewAddrBook(p2pCfg)
	priv := addrbook.GetPrivkey()

	bandwidthTracker := metrics.NewBandwidthCounter()
	host := newHost(mcfg.Port, priv, bandwidthTracker)
	p2p := &P2P{
		host:            host,
		peerInfoManager: manage.NewPeerInfoManager(mgr.Client),
		chainCfg:        chainCfg,
		subCfg:          mcfg,
		p2pCfg:          p2pCfg,
		client:          mgr.Client,
		api:             mgr.SysAPI,
		addrBook:        addrbook,
		mgr:             mgr,
		taskGroup:       &sync.WaitGroup{},
	}
	p2p.discovery = dht.InitDhtDiscovery(p2p.host, p2p.addrBook.AddrsInfo(), p2p.subCfg, p2p.chainCfg.IsTestNet())
	p2p.connManager = manage.NewConnManager(p2p.host, p2p.discovery, bandwidthTracker)
	p2p.db = store.NewDataStore(p2p.subCfg)
	log.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func newHost(port int32, priv p2pcrypto.PrivKey, bandwidthTracker metrics.Reporter) core.Host {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil
	}
	log.Info("NewMulti", "addr", m.String())
	if bandwidthTracker == nil {
		bandwidthTracker = metrics.NewBandwidthCounter()
	}

	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}

	return host
}

func (p *P2P) managePeers() {
	go p.connManager.MonitorAllPeers(p.subCfg.Seeds, p.host)

	for {
		peerlist := p.discovery.RoutingTale()
		log.Info("managePeers", "RoutingTale show peerlist>>>>>>>>>", peerlist,
			"table size", p.discovery.RoutingTableSize())
		if p.isClose() {
			log.Info("managePeers", "p2p", "closed")

			return
		}

		<-time.After(time.Minute * 10)
		//Reflesh addrbook
		peersInfo := p.discovery.FindLocalPeers(p.connManager.FindNearestPeers())
		p.addrBook.SaveAddr(peersInfo)

	}

}

func (p *P2P) StartP2P() {

	//提供给其他插件使用的共享接口
	env := &prototypes.P2PEnv{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManager,
		Discovery:       p.discovery,
		PeerInfoManager: p.peerInfoManager,
		P2PManager:      p.mgr,
		SubConfig:       p.subCfg,
	}

	protocol.Init(env)

	//debug new
	env2 := &protocol.P2PEnv{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManager,
		Discovery:       p.discovery,
		PeerInfoManager: p.peerInfoManager,
		P2PManager:      p.mgr,
		SubConfig:       p.subCfg,
		DB:              p.db,
	}
	p.env = env2
	p2pstore.Init(env2)

	//new
	go p.managePeers()
	go p.handleP2PEvent()
	go p.findLANPeers()

}

//查询本局域网内是否有节点
func (p *P2P) findLANPeers() {
	peerChan, err := p.discovery.FindLANPeers(p.host, "hello-chain33?")
	if err != nil {
		log.Error("findLANPeers", "err", err.Error())
		return
	}

	for neighbors := range peerChan {
		log.Info(">>>>>>>>>>>>>>>>>>>^_^! Well,Let's Play ^_^!<<<<<<<<<<<<<<<<<<<<<<<<<<")
		//发现局域网内的邻居节点
		err := p.host.Connect(context.Background(), neighbors)
		if err != nil {
			log.Error("findLANPeers", "err", err.Error())
			continue
		}

		p.connManager.AddNeighbors(&neighbors)

	}
}

func (p *P2P) handleP2PEvent() {

	p.subChan = p.mgr.PubSub.Sub(p2pmgr.DHTTypeName)
	//TODO, control goroutine num
	for data := range p.subChan {
		if p.isClose() {
			return
		}
		msg, ok := data.(*queue.Message)
		if !ok {
			log.Error("handleP2PEvent", "recv invalid msg, data=", data)
		}
		p.taskGroup.Add(1)
		go func(qmsg *queue.Message) {
			defer p.taskGroup.Done()
			log.Debug("handleP2PEvent", "recv msg ty", qmsg.Ty)
			protocol.HandleEvent(qmsg)

		}(msg)

	}
}

func (p *P2P) CloseP2P() {
	log.Info("p2p closed")
	p.mgr.PubSub.Unsub(p.subChan)
	atomic.StoreInt32(&p.closed, 1)
	p.waitTaskDone()
	p.connManager.Close()
	p.peerInfoManager.Close()
	p.host.Close()
	prototypes.ClearEventHandler()
}

func (p *P2P) isClose() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *P2P) waitTaskDone() {

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		p.taskGroup.Wait()
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second * 20):
		log.Error("waitTaskDone", "err", "20s timeout")
	}
}
