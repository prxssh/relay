package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/prxssh/relay/internal/tracker"
	"github.com/prxssh/relay/internal/utils"
)

// Peer represents an active, established connection to another BitTorrent
// client. It holds the connection itself and state associated with that peer.
type Peer struct {
	// Network address of the remote peer
	Addr string
	// TCP network connection to the peer
	conn net.Conn
	// Represents the pieces that the remote peer has. It's received
	// immediately after the handshake.
	bitfield utils.Bitfield
	// Tracks the choking and interest status between the client and the peer.
	state *peerState
}

// peerState tracks the connection state with a remote peer. This is
// fundamental to the BitTorrent protocol's tit-for-tat mechanism.
type peerState struct {
	// Are we choking the remote peer?
	amChoking bool
	// Are we interested in the remote peer?
	amInterested bool
	// Is the peer choking use?
	peerChoking bool
	// Is the peer interested in use?
	peerInterested bool
}

// PeerConnectOpts provides the necessary information to establish a connection
// and perform a handshake with a remote peer.
type PeerConnectOpts struct {
	InfoHash [sha1.Size]byte
	PeerID   [sha1.Size]byte
	Pieces   int64
}

func ConnectToPeers(
	remotePeers []*tracker.Peer,
	opts *PeerConnectOpts,
) ([]*Peer, error) {
	var wg sync.WaitGroup
	peerChan := make(chan *Peer, len(remotePeers))

	for _, remotePeer := range remotePeers {
		wg.Add(1)

		go func(rp *tracker.Peer) {
			defer wg.Done()

			peer, err := connectToPeer(rp, opts)
			if err != nil {
				return
			}

			go peer.Start()

			peerChan <- peer
		}(remotePeer)
	}
	wg.Wait()
	close(peerChan)

	var connectedPeers []*Peer
	for peer := range peerChan {
		connectedPeers = append(connectedPeers, peer)
	}

	return connectedPeers, nil
}

func (p *Peer) Start() {
	defer p.conn.Close()
	p.readMessages()
}

func (p *Peer) Read() (*message, error) {
	return unmarshalMessage(p.conn)
}

/////////////// Private ///////////////

func connectToPeer(
	remotePeer *tracker.Peer,
	opts *PeerConnectOpts,
) (*Peer, error) {
	addr := fmt.Sprintf("%s:%d", remotePeer.IP, remotePeer.Port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	p := &Peer{
		Addr:     addr,
		conn:     conn,
		state:    initialPeerState(),
		bitfield: utils.NewBitfield(int(opts.Pieces)),
	}

	if err := p.peformHandshake(opts); err != nil {
		return nil, err
	}

	return p, nil
}

func initialPeerState() *peerState {
	return &peerState{
		amChoking:      true,
		amInterested:   false,
		peerChoking:    true,
		peerInterested: false,
	}
}

func (p *Peer) peformHandshake(opts *PeerConnectOpts) error {
	p.conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer p.conn.SetDeadline(time.Time{})

	reqHandshake := newHandshake(opts.InfoHash, opts.PeerID)
	_, err := p.conn.Write(reqHandshake.serialize())
	if err != nil {
		return err
	}

	resHandshake, err := readHanshake(p.conn)
	if err != nil {
		return err
	}

	if !bytes.Equal(resHandshake.infoHash[:], opts.InfoHash[:]) {
		return errors.New("handshake: info hash mismatch")
	}

	if !bytes.Equal(resHandshake.peerID[:], opts.PeerID[:]) {
		return errors.New("handshake: peer id mismatch")
	}

	return nil
}

func (p *Peer) readMessages() {
	for {
		p.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		msg, err := p.Read()
		if err != nil {
			return
		}

		if msg == nil { // keep-alive
			continue
		}

		switch msg.id {
		case msgBitfield:
			p.bitfield = msg.payload

		case msgChoke:
			p.state.peerChoking = true

		case msgUnchoke:
			p.state.peerChoking = false

		case msgInterested:
			p.state.peerInterested = true

		case msgNotInterested:
			p.state.peerInterested = false

		case msgHave:
			// do something

		case msgPiece:
			// do something

		default:
			// raise error/log
		}
	}
}

func (p *Peer) sendMessage(message *message) error {
	_, err := p.conn.Write(message.marshal())
	return err
}
