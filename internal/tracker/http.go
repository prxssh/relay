package tracker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/prxssh/relay/internal/bencode"
)

// HTTPTrackerClient is an HTTP-based implementation of ITrackerProtocol
type HTTPTrackerClient struct {
	announceURL *url.URL
	client      *http.Client
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

func (c *HTTPTrackerClient) Announce(ctx context.Context, params *AnnounceParams) (*AnnounceResponse, error) {
	reqURL := c.buildAnnounceURL(params)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tracker returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseTrackerResponse(resp.Body)
}

// ///////////// Private ///////////////

func newHTTPTrackerClient(url *url.URL) (*HTTPTrackerClient, error) {
	return &HTTPTrackerClient{
		announceURL: url,
		client:      &http.Client{},
	}, nil
}

func (c *HTTPTrackerClient) buildAnnounceURL(params *AnnounceParams) string {
	reqURL := *c.announceURL

	q := reqURL.Query()
	q.Set(paramInfoHash, string(params.InfoHash[:]))
	q.Set(paramPeerID, string(params.PeerID[:]))
	q.Set(paramPort, strconv.Itoa(int(params.Port)))
	q.Set(paramUploaded, strconv.FormatInt(params.Uploaded, 10))
	q.Set(paramDownloaded, strconv.FormatInt(params.Downloaded, 10))
	q.Set(paramLeft, strconv.FormatInt(params.Left, 10))
	q.Set(paramCompact, "1")

	if params.Event != "" {
		q.Set(paramEvent, string(params.Event))
	}

	return reqURL.String()
}

func parseTrackerResponse(r io.Reader) (*AnnounceResponse, error) {
	raw, err := bencode.NewUnmarshaller(r).Unmarshal()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracker response: %w", err)
	}
	data, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type, expected dictionary, got %T", raw)
	}

	if failure, ok := data[keyFailureReason].(string); ok {
		return nil, fmt.Errorf("tracker error: %s", failure)
	}

	if warning, ok := data[keyWarningMsg].(string); ok {
		slog.Warn("Tracker warning", "message", warning)
	}

	getInt64 := func(key string) (int64, bool) {
		val, ok := data[key]
		if !ok {
			return 0, false
		}
		num, ok := val.(int64)
		return num, ok
	}

	interval, ok := getInt64(keyInterval)
	if !ok {
		return nil, fmt.Errorf("tracker response missing or invalid 'interval'")
	}

	// Parse optional fields.
	minInterval, _ := getInt64(keyMinInterval)
	complete, _ := getInt64(keyComplete)
	incomplete, _ := getInt64(keyIncomplete)
	trackerID, _ := data[keyTrackerID].(string)

	peers, err := parsePeers(data)
	if err != nil {
		return nil, err
	}

	return &AnnounceResponse{
		Peers:       peers,
		TrackerID:   trackerID,
		Interval:    uint32(interval),
		Seeders:     uint32(complete),
		Leechers:    uint32(incomplete),
		MinInterval: uint32(minInterval),
	}, nil
}

func parsePeers(data map[string]any) ([]*Peer, error) {
	peersData, ok := data[keyPeers]
	if !ok {
		// It's common for trackers to omit the 'peers' key if there are none.
		// Return an empty slice instead of an error.
		return []*Peer{}, nil
	}

	switch peers := peersData.(type) {
	case string:
		return parseCompactPeers([]byte(peers))
	case []any:
		return parseDictPeers(peers)
	default:
		return nil, fmt.Errorf("invalid 'peers' format: expected string or list, got %T", peersData)
	}
}

func parseCompactPeers(peerData []byte) ([]*Peer, error) {
	const peerSize = 6 // 4 bytes for IP, 2 for port.
	if len(peerData)%peerSize != 0 {
		return nil, fmt.Errorf("invalid compact peer list length: %d", len(peerData))
	}

	numPeers := len(peerData) / peerSize
	peers := make([]*Peer, 0, numPeers)

	for i := 0; i < len(peerData); i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peerData[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peerData[offset+4 : offset+6])
	}
	return peers, nil
}

func parseDictPeers(peerList []any) ([]*Peer, error) {
	peers := make([]*Peer, 0, len(peerList)) // Pre-allocate slice capacity.

	for i, item := range peerList {
		peerDict, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid peer dictionary entry at index %d: got %T", i, item)
		}

		ipStr, ok := peerDict[keyPeerIP].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'ip' in peer entry at index %d", i)
		}

		portVal, ok := peerDict[keyPeerPort].(int64)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'port' in peer entry at index %d", i)
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address string '%s' in peer entry at index %d", ipStr, i)
		}

		peer := &Peer{
			IP:   ip,
			Port: uint16(portVal),
		}
		// Peer ID is optional.
		if id, ok := peerDict[keyPeerID].(string); ok {
			peer.ID = id
		}

		peers = append(peers, peer)
	}
	return peers, nil
}
