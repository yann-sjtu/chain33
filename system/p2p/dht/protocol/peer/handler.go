package peer

import (
	"encoding/json"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) handleStreamPeerInfo(stream network.Stream) {
	peerInfo := p.getLocalPeerInfo()
	err := protocol.WriteStream(peerInfo, stream)
	if err != nil {
		log.Error("handleStreamPeerInfo", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleStreamVersion(stream network.Stream) {
	var req types.P2PVersion
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("handleStreamVersion", "read stream error", err)
		return
	}

	if ip, _ := parseIPAndPort(req.GetAddrFrom()); isPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(req.GetAddrFrom())
		if err != nil {
			return
		}
		p.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), remoteMAddr, time.Hour*24)
	}

	p.setExternalAddr(req.GetAddrRecv())
	resp := &types.P2PVersion{
		AddrFrom:  p.getExternalAddr(),
		AddrRecv:  stream.Conn().RemoteMultiaddr().String(),
		Timestamp: time.Now().Unix(),
	}
	err = protocol.WriteStream(resp, stream)
	if err != nil {
		log.Error("handleStreamVersion", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleEventPeerInfo(msg *queue.Message) {
	peers := p.PeerInfoManager.FetchAll()
	msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))
}

func (p *Protocol) handleEventNetProtocols(msg *queue.Message) {
	//all protocols net info
	bandProtocols := p.ConnManager.BandTrackerByProtocol()
	allProtocolNetInfo, _ := json.MarshalIndent(bandProtocols, "", "\t")
	log.Debug("handleEventNetInfo", "allProtocolNetInfo", string(allProtocolNetInfo))
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventNetProtocols, bandProtocols))
}

func (p *Protocol) handleEventNetInfo(msg *queue.Message) {
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo

	netinfo.Externaladdr = p.getPublicIP()
	localips, _ := localIPv4s()
	if len(localips) != 0 {
		log.Error("handleEventNetInfo", "localIps", localips)
		netinfo.Localaddr = localips[0]
	} else {
		netinfo.Localaddr = netinfo.Externaladdr
	}

	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	netinfo.Peerstore = int32(len(p.Host.Peerstore().PeersWithAddrs()))
	netinfo.Routingtable = int32(p.RoutingTable.Size())
	netstat := p.ConnManager.GetNetRate()

	netinfo.Ratein = p.ConnManager.RateCalculate(netstat.RateIn)
	netinfo.Rateout = p.ConnManager.RateCalculate(netstat.RateOut)
	netinfo.Ratetotal = p.ConnManager.RateCalculate(netstat.RateOut + netstat.RateIn)
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReplyNetInfo, &netinfo))
}