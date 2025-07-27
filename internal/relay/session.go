package relay

import (
	"context"
	"crypto/sha1"
	"errors"
	"sync"
	"time"

	"github.com/prxssh/relay/internal/torrent"
	"github.com/prxssh/relay/internal/tracker"
)

// torrentStatus represents the various states a torrent session can be in.
type torrentStatus string

// managedTracker wraps a tracker client with its specific state, such as its
// personal announce interval and the time for its next announce.
type managedTracker struct {
	client           tracker.ITrackerProtocol
	interval         time.Duration
	nextAnnounceTime time.Time
	failures         int
	isAnnouncing     bool
}

// session represents the state and metadata for an active torrent
// download. It holds all the necessary information to mangae the lifecycle of
// a torrent, from communicating with the tracker to tracking download
// upload/progress.
type session struct {
	// Unique 20-byte ID for this client
	peerID [sha1.Size]byte
	// Parsed data from the .torrent file
	torrent *torrent.Torrent
	// Client used to communicate with tracker
	trackers []*managedTracker
	mu       sync.Mutex
	// Duration the client should wait between tracker announce
	announceInterval time.Duration
	// Indicates the current state of the torrent download
	status torrentStatus
	// Total number of bytes downloaded till now
	downloaded int64
	// Total number of bytes uploaded till now
	uploaded   int64
	ctx        context.Context
	cancelFunc context.CancelFunc
}

const (
	statusStarted    torrentStatus = "started"
	statusPaused     torrentStatus = "paused"
	statusCompleted  torrentStatus = "completed"
	statusStopped    torrentStatus = "stopped"
	statusInProgress torrentStatus = "in-progress"
)

const defaultAnnounceInterval = 30 * time.Minute

func newSession(parentCtx context.Context, clientID [sha1.Size]byte, torrent *torrent.Torrent) (*session, error) {
	ctx, cancelFunc := context.WithCancel(parentCtx)

	var managedTrackers []*managedTracker
	for _, url := range torrent.AnnounceURLs {
		trackerClient, err := tracker.New(url)
		if err != nil {
			continue
		}
		managedTrackers = append(managedTrackers, &managedTracker{
			client:           trackerClient,
			interval:         defaultAnnounceInterval,
			nextAnnounceTime: time.Now(),
		})
	}

	if len(managedTrackers) == 0 {
		cancelFunc()
		return nil, errors.New("failed to initialize any trackers")
	}

	return &session{
		peerID:     clientID,
		torrent:    torrent,
		trackers:   managedTrackers,
		status:     statusStarted,
		downloaded: 0,
		uploaded:   0,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}, nil
}

func (s *session) Start() {
	go s.announceLoop()
}

func (s *session) Stop() {
	s.cancelFunc()
}

/////////////// Private ///////////////

func (s *session) announceLoop() {
	s.broadcastAnnounce(statusStarted)
	defer s.broadcastAnnounce(statusStopped)

	for {
		var nextAnnounceTime *time.Time
		s.mu.Lock()
		for _, mt := range s.trackers {
			if !mt.isAnnouncing && (nextAnnounceTime == nil || mt.nextAnnounceTime.Before(*nextAnnounceTime)) {
				t := mt.nextAnnounceTime
				nextAnnounceTime = &t
			}
		}
		s.mu.Unlock()

		waitDuration := defaultAnnounceInterval
		if nextAnnounceTime != nil {
			waitDuration = time.Until(*nextAnnounceTime)
		}

		timer := time.NewTimer(waitDuration)

		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			now := time.Now()
			s.mu.Lock()
			for _, mt := range s.trackers {
				if !mt.isAnnouncing && now.After(mt.nextAnnounceTime) {
					mt.isAnnouncing = true
					go s.announceToTracker(mt, s.status)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *session) announceToTracker(mt *managedTracker, event torrentStatus) {
	defer func() {
		s.mu.Lock()
		mt.isAnnouncing = false
		s.mu.Unlock()
	}()

	s.mu.Lock()
	req := &tracker.AnnounceParams{
		InfoHash:   s.torrent.Info.Hash,
		PeerID:     s.peerID,
		Downloaded: s.downloaded,
		Uploaded:   s.uploaded,
		Left:       s.torrent.Size - s.downloaded,
		Port:       6969,
		Event:      toTrackerStatus(event),
	}
	s.mu.Unlock()

	res, err := mt.client.Announce(s.ctx, req)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		mt.failures++
		backoffInterval := mt.interval * time.Duration(mt.failures+1)
		mt.nextAnnounceTime = time.Now().Add(backoffInterval)
		return
	}

	mt.failures = 0
	mt.interval = time.Duration(res.Interval) * time.Second
	if mt.interval <= 0 {
		mt.interval = defaultAnnounceInterval
	}
	mt.nextAnnounceTime = time.Now().Add(mt.interval)
}

func (s *session) broadcastAnnounce(event torrentStatus) {
	s.mu.Lock()
	// Copy the slice of trackers to avoid race conditions during iteration.
	trackers := make([]*managedTracker, len(s.trackers))
	copy(trackers, s.trackers)
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, mt := range trackers {
		wg.Add(1)
		go func(tracker *managedTracker) {
			defer wg.Done()
			s.announceToTracker(tracker, event)
		}(mt)
	}
	wg.Wait()
}

func toTrackerStatus(event torrentStatus) tracker.Event {
	if event == statusStopped {
		return tracker.EventStopped
	} else if event == statusCompleted {
		return tracker.EventCompleted
	} else {
		return tracker.EventStarted
	}
}
