package relay

import (
	"crypto/sha1"
	"time"

	"github.com/prxssh/relay/internal/torrent"
	"github.com/prxssh/relay/internal/tracker"
)

// torrentStatus represents the various states a torrent session can be in.
type torrentStatus string

const (
	statusStarted    torrentStatus = "started"
	statusPaused     torrentStatus = "paused"
	statusCompleted  torrentStatus = "completed"
	statusCancelled  torrentStatus = "cancelled"
	statusInProgress torrentStatus = "in-progress"
)

// torrentSession represents the state and metadata for an active torrent
// download. It holds all the necessary information to mangae the lifecycle of
// a torrent, from communicating with the tracker to tracking download
// upload/progress.
type torrentSession struct {
	// Unique 20-byte ID for this client
	peerID [sha1.Size]byte
	// Parsed data from the .torrent file
	metainfo *torrent.Torrent
	// Client used to communicate with tracker
	tracker tracker.ITrackerProtocol
	// Duration the client should wait between tracker announce
	announceInterval time.Duration
	// Indicates the current state of the torrent download
	status torrentStatus
	// Total number of bytes downloaded till now
	downloaded int64
	// Total number of bytes uploaded till now
	uploaded int64
}

func NewSession(clientID [sha1.Size]byte, metainfo *torrent.Torrent) (*torrentSession, error) {
	return &torrentSession{
		peerID:     clientID,
		metainfo:   metainfo,
		status:     statusStarted,
		downloaded: 0,
		uploaded:   0,
	}, nil
}
