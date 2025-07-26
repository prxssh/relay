package torrent

import (
	"bytes"
	"os"
)

func ReadTorrentFromPath(path string) (*Metainfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return New(bytes.NewReader(data))
}
