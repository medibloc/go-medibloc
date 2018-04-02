package net

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/medibloc/go-medibloc/util/byteutils"

	"github.com/gogo/protobuf/proto"
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	netpb "github.com/medibloc/go-medibloc/net/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Stream Message Type
const (
	ClientVersion = "0.3.0"
	MedProtocolID = "/med/1.0.0"
	HELLO         = "hello"
	OK            = "ok"
	BYE           = "bye"
	SYNCROUTE     = "syncroute"
	ROUTETABLE    = "routetable"
	RECVEDMSG     = "recvedmsg"
)

// Stream Status
const (
	streamStatusInit = iota
	streamStatusHandshakeSucceed
	streamStatusClosed
)

// Stream Errors
var (
	ErrShouldCloseConnectionAndExitLoop = errors.New("should close connection and exit loop")
	ErrStreamIsNotConnected             = errors.New("stream is not connected")
)

// Stream define the structure of a stream in p2p network
type Stream struct {
	syncMutex                 sync.Mutex
	pid                       peer.ID
	addr                      ma.Multiaddr
	stream                    libnet.Stream
	node                      *Node
	handshakeSucceedCh        chan bool
	messageNotifChan          chan int
	highPriorityMessageChan   chan *MedMessage
	normalPriorityMessageChan chan *MedMessage
	lowPriorityMessageChan    chan *MedMessage
	quitWriteCh               chan bool
	status                    int
	connectedAt               int64
	latestReadAt              int64
	latestWriteAt             int64
	msgCount                  map[string]int
}

// NewStream return a new Stream
func NewStream(stream libnet.Stream, node *Node) *Stream {
	return newStreamInstance(stream.Conn().RemotePeer(), stream.Conn().RemoteMultiaddr(), stream, node)
}

// NewStreamFromPID return a new Stream based on the pid
func NewStreamFromPID(pid peer.ID, node *Node) *Stream {
	return newStreamInstance(pid, nil, nil, node)
}

func newStreamInstance(pid peer.ID, addr ma.Multiaddr, stream libnet.Stream, node *Node) *Stream {
	return &Stream{
		pid:                       pid,
		addr:                      addr,
		stream:                    stream,
		node:                      node,
		handshakeSucceedCh:        make(chan bool, 1),
		messageNotifChan:          make(chan int, 6*1024),
		highPriorityMessageChan:   make(chan *MedMessage, 2*1024),
		normalPriorityMessageChan: make(chan *MedMessage, 2*1024),
		lowPriorityMessageChan:    make(chan *MedMessage, 2*1024),
		quitWriteCh:               make(chan bool, 1),
		status:                    streamStatusInit,
		connectedAt:               time.Now().Unix(),
		latestReadAt:              0,
		latestWriteAt:             0,
		msgCount:                  make(map[string]int),
	}
}

// Connect to the stream
func (s *Stream) Connect() error {
	logging.WithFields(logrus.Fields{
		"stream": s.String(),
	}).Debug("Connecting to peer.")

	// connect to host.
	stream, err := s.node.host.NewStream(
		s.node.context,
		s.pid,
		MedProtocolID,
	)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"stream": s.String(),
			"err":    err,
		}).Debug("Failed to connect to host.")
		return err
	}
	s.stream = stream
	s.addr = stream.Conn().RemoteMultiaddr()

	return nil
}

// IsConnected return if the stream is connected
func (s *Stream) IsConnected() bool {
	return s.stream != nil
}

// IsHandshakeSucceed return if the handshake in the stream succeed
func (s *Stream) IsHandshakeSucceed() bool {
	return s.status == streamStatusHandshakeSucceed
}

func (s *Stream) String() string {
	addrStr := ""
	if s.addr != nil {
		addrStr = s.addr.String()
	}

	return fmt.Sprintf("Peer Stream: %s,%s", s.pid.Pretty(), addrStr)
}

// SendProtoMessage send proto msg to buffer
func (s *Stream) SendProtoMessage(messageName string, pb proto.Message, priority int) error {
	data, err := proto.Marshal(pb)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":         err,
			"messageName": messageName,
			"stream":      s.String(),
		}).Debug("Failed to marshal proto message.")
		return err
	}

	return s.SendMessage(messageName, data, priority)
}

// SendMessage send msg to buffer
func (s *Stream) SendMessage(messageName string, data []byte, priority int) error {
	message, err := NewMedMessage(s.node.config.ChainID, DefaultReserved, 0, messageName, data)
	if err != nil {
		return err
	}

	// metrics.
	metricsPacketsOutByMessageName(messageName, message.Length())

	// send to pool.
	message.FlagSendMessageAt()

	// use a non-blocking channel to avoid blocking when the channel is full.
	switch priority {
	case MessagePriorityHigh:
		s.highPriorityMessageChan <- message
	case MessagePriorityNormal:
		select {
		case s.normalPriorityMessageChan <- message:
		default:
			logging.WithFields(logrus.Fields{
				"normalPriorityMessageChan.len": len(s.normalPriorityMessageChan),
				"stream":                        s.String(),
			}).Debug("Received too many normal priority message.")
			return nil
		}
	default:
		select {
		case s.lowPriorityMessageChan <- message:
		default:
			logging.WithFields(logrus.Fields{
				"lowPriorityMessageChan.len": len(s.lowPriorityMessageChan),
				"stream":                     s.String(),
			}).Debug("Received too many low priority message.")
			return nil
		}
	}
	select {
	case s.messageNotifChan <- 1:
	default:
		logging.WithFields(logrus.Fields{
			"messageNotifChan.len": len(s.messageNotifChan),
			"stream":               s.String(),
		}).Debug("Received too many message notifChan.")
		return nil
	}
	return nil
}

func (s *Stream) Write(data []byte) error {
	if s.stream == nil {
		s.Close(ErrStreamIsNotConnected)
		return ErrStreamIsNotConnected
	}

	// at least 5kb/s to write message
	deadline := time.Now().Add(time.Duration(len(data)/1024/5+1) * time.Second)
	if err := s.stream.SetWriteDeadline(deadline); err != nil {
		return err
	}
	n, err := s.stream.Write(data)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":    err,
			"stream": s.String(),
		}).Warn("Failed to send message to peer.")
		s.Close(err)
		return err
	}
	s.latestWriteAt = time.Now().Unix()

	// metrics.
	metricsPacketsOut.Mark(1)
	metricsBytesOut.Mark(int64(n))

	return nil
}

// WriteMedMessage write med msg in the stream
func (s *Stream) WriteMedMessage(message *MedMessage) error {
	// metrics.
	metricsPacketsOutByMessageName(message.MessageName(), message.Length())

	err := s.Write(message.Content())
	message.FlagWriteMessageAt()

	/*
		// debug logs.
		logging.WithFields(logrus.Fields{
			"stream":      s.String(),
			"err":         err,
			"checksum":    message.DataCheckSum(),
			"messageName": message.MessageName(),
			"latency(ms)": message.LatencyFromSendToWrite(),
		}).Debugf("Written %s message to peer.", message.MessageName())
	*/

	return err
}

// WriteProtoMessage write proto msg in the stream
func (s *Stream) WriteProtoMessage(messageName string, pb proto.Message) error {
	data, err := proto.Marshal(pb)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":         err,
			"messageName": messageName,
			"stream":      s.String(),
		}).Debug("Failed to marshal proto message.")
		return err
	}

	return s.WriteMessage(messageName, data)
}

// WriteMessage write raw msg in the stream
func (s *Stream) WriteMessage(messageName string, data []byte) error {
	message, err := NewMedMessage(s.node.config.ChainID, DefaultReserved, 0, messageName, data)
	if err != nil {
		return err
	}

	return s.WriteMedMessage(message)
}

// StartLoop start stream handling loop.
func (s *Stream) StartLoop() {
	go s.writeLoop()
	go s.readLoop()
}

func (s *Stream) readLoop() {
	// send Hello to host if stream is not connected.
	if !s.IsConnected() {
		if err := s.Connect(); err != nil {
			s.Close(err)
			return
		}
		if err := s.Hello(); err != nil {
			s.Close(err)
			return
		}
	}

	// loop.
	buf := make([]byte, 1024*4)
	messageBuffer := make([]byte, 0)

	var message *MedMessage

	for {
		n, err := s.stream.Read(buf)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err":    err,
				"stream": s.String(),
			}).Debug("Error occurred when reading data from network connection.")
			if err != io.EOF {
				s.Close(err)
			}
			return
		}

		messageBuffer = append(messageBuffer, buf[:n]...)
		s.latestReadAt = time.Now().Unix()

		for {
			if message == nil {
				var err error

				// waiting for header data.
				if len(messageBuffer) < MedMessageHeaderLength {
					// continue reading.
					break
				}

				message, err = ParseMedMessage(messageBuffer)
				if err != nil {
					s.Bye()
					return
				}

				// check ChainID.
				if s.node.config.ChainID != message.ChainID() {
					logging.WithFields(logrus.Fields{
						"err":             err,
						"stream":          s.String(),
						"conf.chainID":    s.node.config.ChainID,
						"message.chainID": message.ChainID(),
					}).Warn("Invalid chainID, disconnect the connection.")
					s.Bye()
					return
				}

				// remove header from buffer.
				messageBuffer = messageBuffer[MedMessageHeaderLength:]
			}

			// waiting for data.
			if len(messageBuffer) < int(message.DataLength()) {
				// continue reading.
				break
			}

			if err := message.ParseMessageData(messageBuffer); err != nil {
				s.Bye()
				return
			}

			// remove data from buffer.
			messageBuffer = messageBuffer[message.DataLength():]

			/*
				nowAt := time.Now().UnixNano()
				logging.WithFields(logrus.Fields{
					"messageName":                    message.MessageName(),
					"checksum":                       message.DataCheckSum(),
					"stream":                         s.String(),
					"duration from Read(ms)":         (nowAt - readDataAt) / int64(time.Millisecond),
					"duration from last Message(ms)": (nowAt - readAt) / int64(time.Millisecond),
					"len(messageBuffer)":             len(messageBuffer),
				}).Debugf("Received %s message from peer.", message.MessageName())
			*/
			// metrics.
			metricsPacketsIn.Mark(1)
			metricsBytesIn.Mark(int64(message.Length()))
			metricsPacketsInByMessageName(message.MessageName(), message.Length())

			// handle message.
			if err := s.handleMessage(message); err == ErrShouldCloseConnectionAndExitLoop {
				s.Bye()
				return
			}

			// reset message.
			message = nil
		}
	}
}

func (s *Stream) writeLoop() {
	// waiting for handshake succeed.
	handshakeTimeoutTicker := time.NewTicker(30 * time.Second)
	select {
	case <-s.handshakeSucceedCh:
		// handshake succeed.
	case <-s.quitWriteCh:
		logging.WithFields(logrus.Fields{
			"stream": s.String(),
		}).Debug("Quiting Stream Write Loop.")
		return
	case <-handshakeTimeoutTicker.C:
		logging.WithFields(logrus.Fields{
			"stream": s.String(),
		}).Debug("Handshaking Stream timeout, quiting.")
		s.Close(errors.New("Handshake timeout"))
		return
	}

	for {
		select {
		case <-s.quitWriteCh:
			logging.WithFields(logrus.Fields{
				"stream": s.String(),
			}).Debug("Quiting Stream Write Loop.")
			return
		case <-s.messageNotifChan:
			select {
			case message := <-s.highPriorityMessageChan:
				s.WriteMedMessage(message)
				continue
			default:
			}

			select {
			case message := <-s.normalPriorityMessageChan:
				s.WriteMedMessage(message)
				continue
			default:
			}

			select {
			case message := <-s.lowPriorityMessageChan:
				s.WriteMedMessage(message)
				continue
			default:
			}
		}
	}
}

func (s *Stream) handleMessage(message *MedMessage) error {
	messageName := message.MessageName()
	s.msgCount[messageName]++

	switch messageName {
	case HELLO:
		return s.onHello(message)
	case OK:
		return s.onOk(message)
	case BYE:
		return s.onBye(message)
	}

	// check handshake status.
	if s.status != streamStatusHandshakeSucceed {
		return ErrShouldCloseConnectionAndExitLoop
	}

	switch messageName {
	case SYNCROUTE:
		return s.onSyncRoute(message)
	case ROUTETABLE:
		return s.onRouteTable(message)
	case RECVEDMSG:
		return s.onRecvedMsg(message)
	default:
		s.node.netService.PutMessage(NewBaseMessage(message.MessageName(), s.pid.Pretty(), message.Data()))
		// record recv message.
		RecordRecvMessage(s, message.DataCheckSum())
	}

	return nil
}

// Close close the stream
func (s *Stream) Close(reason error) {
	// Add lock & close flag to prevent multi call.
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	if s.status == streamStatusClosed {
		return
	}
	s.status = streamStatusClosed

	logging.WithFields(logrus.Fields{
		"stream": s.String(),
		"reason": reason,
	}).Debug("Closing stream.")

	// cleanup.
	s.node.streamManager.RemoveStream(s)
	s.node.routeTable.RemovePeerStream(s)

	// quit.
	s.quitWriteCh <- true

	// close stream.
	if s.stream != nil {
		s.stream.Close()
	}
}

// Bye say bye in the stream
func (s *Stream) Bye() {
	s.WriteMessage(BYE, []byte{})
	s.Close(errors.New("bye: force close"))
}

func (s *Stream) onBye(message *MedMessage) error {
	logging.WithFields(logrus.Fields{
		"stream": s.String(),
	}).Debug("Received Bye message, close the connection.")
	return ErrShouldCloseConnectionAndExitLoop
}

// Hello say hello in the stream
func (s *Stream) Hello() error {
	msg := &netpb.Hello{
		NodeId:        s.node.id.String(),
		ClientVersion: ClientVersion,
	}
	return s.WriteProtoMessage(HELLO, msg)
}

func (s *Stream) onHello(message *MedMessage) error {
	msg, err := netpb.HelloMessageFromProto(message.Data())
	if err != nil {
		return ErrShouldCloseConnectionAndExitLoop
	}

	if msg.NodeId != s.pid.String() || !CheckClientVersionCompatibility(ClientVersion, msg.ClientVersion) {
		// invalid client, bye().
		logging.WithFields(logrus.Fields{
			"pid":               s.pid.Pretty(),
			"address":           s.addr,
			"ok.node_id":        msg.NodeId,
			"ok.client_version": msg.ClientVersion,
		}).Warn("Invalid NodeId or incompatible client version.")
		return ErrShouldCloseConnectionAndExitLoop
	}

	// add to route table.
	s.node.routeTable.AddPeerStream(s)

	// handshake finished.
	s.finishHandshake()

	return s.Ok()
}

// Ok say ok in the stream
func (s *Stream) Ok() error {
	// send OK.
	resp := &netpb.OK{
		NodeId:        s.node.id.String(),
		ClientVersion: ClientVersion,
	}

	return s.WriteProtoMessage(OK, resp)
}

func (s *Stream) onOk(message *MedMessage) error {
	msg, err := netpb.OKMessageFromProto(message.Data())
	if err != nil {
		return ErrShouldCloseConnectionAndExitLoop
	}

	if msg.NodeId != s.pid.String() || !CheckClientVersionCompatibility(ClientVersion, msg.ClientVersion) {
		// invalid client, bye().
		logging.WithFields(logrus.Fields{
			"pid":               s.pid.Pretty(),
			"address":           s.addr,
			"ok.node_id":        msg.NodeId,
			"ok.client_version": msg.ClientVersion,
		}).Warn("Invalid NodeId or incompatible client version.")
		return ErrShouldCloseConnectionAndExitLoop
	}

	// add to route table.
	s.node.routeTable.AddPeerStream(s)

	// handshake finished.
	s.finishHandshake()

	return nil
}

// SyncRoute send sync route request
func (s *Stream) SyncRoute() error {
	return s.SendMessage(SYNCROUTE, []byte{}, MessagePriorityHigh)
}

func (s *Stream) onSyncRoute(message *MedMessage) error {
	return s.RouteTable()
}

// RouteTable send sync table request
func (s *Stream) RouteTable() error {
	// get random peers from routeTable
	peers := s.node.routeTable.GetRandomPeers(s.pid)

	// prepare the protobuf message.
	msg := &netpb.Peers{
		Peers: make([]*netpb.PeerInfo, len(peers)),
	}

	for i, v := range peers {
		pi := &netpb.PeerInfo{
			Id:    v.ID.Pretty(),
			Addrs: make([]string, len(v.Addrs)),
		}
		for j, addr := range v.Addrs {
			pi.Addrs[j] = addr.String()
		}
		msg.Peers[i] = pi
	}

	logging.WithFields(logrus.Fields{
		"stream":          s.String(),
		"routetableCount": len(peers),
	}).Debug("Replied sync route message.")

	return s.SendProtoMessage(ROUTETABLE, msg, MessagePriorityHigh)
}

func (s *Stream) onRouteTable(message *MedMessage) error {
	peers := new(netpb.Peers)
	if err := proto.Unmarshal(message.Data(), peers); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Invalid Peers proto message.")
		return ErrShouldCloseConnectionAndExitLoop
	}

	s.node.routeTable.AddPeers(s.node.ID(), peers)

	return nil
}

// RecvedMsg send received msg
func (s *Stream) RecvedMsg(hash uint32) error {
	return s.SendMessage(RECVEDMSG, byteutils.FromUint32(hash), MessagePriorityHigh)
}

func (s *Stream) onRecvedMsg(message *MedMessage) error {
	hash := byteutils.Uint32(message.Data())
	RecordRecvMessage(s, hash)

	return nil
}

func (s *Stream) finishHandshake() {
	logging.WithFields(logrus.Fields{
		"stream": s.String(),
	}).Debug("Finished handshake.")

	s.status = streamStatusHandshakeSucceed
	s.handshakeSucceedCh <- true
}

// CheckClientVersionCompatibility if two clients are compatible
func CheckClientVersionCompatibility(v1, v2 string) bool {
	return v1 == v2
}
