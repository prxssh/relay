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
	// SHA1 hash of the info key
	InfoHash [sha1.Size]byte
	// Echo client PeerID
	PeerID [sha1.Size]byte
	// Port on which we're listening for connections
	Port uint16
	// Data that has been seeded so far.
	Uploaded int64
	// Data that has been downloaded so far.
	Downloaded int64
	// Data left to download
	Left int64
	// Current event (started/completed/stopped)
	Event Event
}

// AnnounceResponse is what the tracker returns on announce
type AnnounceResponse struct {
	// Unique identifier for the tracker
	TrackerID string
	// Seconds until next announce
	Interval uint32
	// Clients downloading this torrent
	Leechers uint32
	// Clients uploading this torrent
	Seeders uint32
	// Active peers
	Peers []*Peer
	// Interval after which we should call the tracker
	MinInterval uint32
}

// Peer is one peer endpoint from the tracker
type Peer struct {
	// Identifier for this peer (absent in compact mode)
	ID string
	// IP of this peer
	IP net.IP
	// Port on which this peer is listenting to connections
	Port uint16
}

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
