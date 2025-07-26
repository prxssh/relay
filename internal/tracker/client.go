package tracker

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"net/url"
)

// ITrackerProtocol defines the standard Tracker operations
type ITrackerProtocol interface {
	// Announce sends the client's state to the tracker and returns the
	// tracker's response
	Announce(ctx context.Context, params *AnnounceParams) (*AnnounceResponse, error)
}

type Event string

const (
	EventStarted   Event = "started"
	EventCompleted Event = "completed"
	EventStopped   Event = "stopped"
)

// AnnounceParams holds all the fields the tracker needs
type AnnounceParams struct {
	InfoHash   [sha1.Size]byte // SHA1 hash of the info key
	PeerID     [sha1.Size]byte // Echo client PeerID
	Port       uint16          // Port on which we're listening for connections
	Uploaded   int64           // data that has been seeded so far
	Downloaded int64           // data that has been downloaded so far
	Left       int64           // data left to download
	Event      Event           // current event (started/completed/stopped)
}

// AnnounceResponse is what the tracker returns on announce
type AnnounceResponse struct {
	TrackerID   string         // unique identifier for the tracker
	Interval    uint32         // seconds until next announce
	Leechers    uint32         // clients downloading this torrent
	Seeders     uint32         // clients uploading this torrent
	Peers       []*TrackerPeer // active peers
	MinInterval uint32         // interval after which we should call the tracker
}

// Peer is one peer endpoint from the tracker
type TrackerPeer struct {
	ID   string // identifier for this peer (absent in compact mode)
	IP   net.IP // ip-address of this peer
	Port uint16 // port on which this peer is listenting to connections
}

// Constants for tracker requests and responses to avoid "magic strings".
const (
	// Query parameters
	paramInfoHash   = "info_hash"
	paramPeerID     = "peer_id"
	paramPort       = "port"
	paramUploaded   = "uploaded"
	paramDownloaded = "downloaded"
	paramLeft       = "left"
	paramCompact    = "compact"
	paramEvent      = "event"

	// Bencode dictionary keys
	keyFailureReason = "failure reason"
	keyWarningMsg    = "warning message"
	keyInterval      = "interval"
	keyMinInterval   = "min interval"
	keyTrackerID     = "tracker id"
	keyComplete      = "complete"
	keyIncomplete    = "incomplete"
	keyPeers         = "peers"
	keyPeerID        = "peer id"
	keyPeerIP        = "ip"
	keyPeerPort      = "port"
)

func NewTrackerClient(announce string) (ITrackerProtocol, error) {
	u, err := url.Parse(announce)
	if err != nil {
		return nil, fmt.Errorf("tracker: invalid announce %q:%w", announce, err)
	}

	switch u.Scheme {
	case "http", "https":
		return newHTTPTrackerClient(u)
	default:
		return nil, fmt.Errorf("tracker: unsupported tracker protocol %q", u.Scheme)
	}
}
