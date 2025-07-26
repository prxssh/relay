package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"

	"github.com/prxssh/relay/internal/bencode"
)

// File represents a single file within a multi-file torrent
type File struct {
	Length int64
	MD5    string
	Path   []string
}

// Info contains the file-specific information of the torrent.
// This is the part of the metainfo that gets hashed to create the InfoHash
type Info struct {
	Name      string
	PieceLen  int64
	Pieces    [][sha1.Size]byte
	IsPrivate bool
	// For single-file mode
	Length int64
	// For multi-file mode
	Files []*File
	// SHA1 hash of the info dict
	Hash [sha1.Size]byte
}

// Metainfo represents the complete data from a .torernt file
type Metainfo struct {
	AnnounceURLs []string
	CreationDate int64
	Comment      string
	CreatedBy    string
	Encoding     string
	Info         *Info
	Size         int64
}

func (m *Metainfo) NumPieces() int {
	return (len(m.Info.Pieces) + 7) / 8
}

func New(r io.Reader) (*Metainfo, error) {
	p, err := newParser(r)
	if err != nil {
		return nil, err
	}
	return p.parse()
}

/////////////// Private ///////////////

func (i *Info) Size() int64 {
	if len(i.Files) == 0 {
		return i.Length
	}

	var size int64
	for _, f := range i.Files {
		size += f.Length
	}

	return size
}

type parser struct {
	data map[string]any
}

func newParser(r io.Reader) (*parser, error) {
	unmarshalled, err := bencode.NewUnmarshaller(r).Unmarshal()
	if err != nil {
		return nil, err
	}

	data, ok := unmarshalled.(map[string]any)
	if !ok {
		return nil, errors.New("metainfo: top-level is not a dictionary")
	}

	return &parser{data: data}, nil
}

func (p *parser) parse() (*Metainfo, error) {
	info, err := p.parseInfo()
	if err != nil {
		return nil, fmt.Errorf("metainfo: failed to parse info dict: %w", err)
	}

	announceURLs, err := p.parseAnnounce()
	if err != nil {
		return nil, err
	}

	return &Metainfo{
		Info:         info,
		AnnounceURLs: announceURLs,
		CreationDate: p.getInt("creation date"),
		Comment:      p.getString("comment"),
		CreatedBy:    p.getString("created by"),
		Size:         info.Size(),
	}, nil
}

func (p *parser) parseInfo() (*Info, error) {
	infoDict, ok := p.data["info"].(map[string]any)
	if !ok {
		return nil, errors.New("'info' key is missing or not a dictionary")
	}

	infoHash, err := calculateSHA1Hash(infoDict)
	if err != nil {
		return nil, err
	}

	infoParser := &parser{data: infoDict}

	piecesStr, ok := infoParser.data["pieces"].(string)
	if !ok {
		return nil, errors.New("'pieces' key is missing or not a string")
	}
	if len(piecesStr)%sha1.Size != 0 {
		return nil, fmt.Errorf("invalid pieces length %d", len(piecesStr))
	}
	pieces := make([][sha1.Size]byte, len(piecesStr)/sha1.Size)
	for i := 0; i < len(pieces); i++ {
		copy(pieces[i][:], piecesStr[i*sha1.Size:])
	}

	files, err := infoParser.parseFiles()
	if err != nil {
		return nil, err
	}

	return &Info{
		Hash:      infoHash,
		Name:      infoParser.getString("name"),
		PieceLen:  infoParser.getInt("piece length"),
		Pieces:    pieces,
		IsPrivate: infoParser.getInt("private") == 1,
		Length:    infoParser.getInt("length"),
		Files:     files,
	}, nil
}

func (p *parser) parseFiles() ([]*File, error) {
	rawFiles, ok := p.data["files"].([]any)
	if !ok {
		return []*File{}, nil // Optional, only for multi-file torrents
	}

	files := make([]*File, 0, len(rawFiles))
	for _, entry := range rawFiles {
		fileDict, ok := entry.(map[string]any)
		if !ok {
			return nil, errors.New("file entry is not a dictionary")
		}
		fileParser := &parser{data: fileDict}

		rawPath, ok := fileDict["path"].([]any)
		if !ok {
			return nil, errors.New("file 'path' is not a list")
		}
		path := make([]string, len(rawPath))
		for i, pth := range rawPath {
			path[i], _ = pth.(string)
		}

		files = append(files, &File{
			Length: fileParser.getInt("length"),
			MD5:    fileParser.getString("md5sum"),
			Path:   path,
		})

	}

	return files, nil
}

func (p *parser) parseAnnounce() ([]string, error) {
	urls := make(map[string]struct{})

	announce := p.getString("announce")
	if announce == "" {
		return nil, errors.New("announce is missing")
	}

	if rawList, ok := p.data["announce-list"].([]any); ok {
		for _, tier := range rawList {
			if tierList, ok := tier.([]any); ok {
				for _, u := range tierList {
					if urlStr, ok := u.(string); ok {
						urls[urlStr] = struct{}{}
					}
				}
			}
		}
	}

	announceList := make([]string, 0, len(urls))
	for u := range urls {
		announceList = append(announceList, u)
	}

	return announceList, nil
}

func (p *parser) getString(key string) string {
	if val, ok := p.data[key].(string); ok {
		return val
	}

	return ""
}

func (p *parser) getInt(key string) int64 {
	if val, ok := p.data[key].(int64); ok {
		return val
	}

	return 0
}

func calculateSHA1Hash(infoDict map[string]any) ([sha1.Size]byte, error) {
	var buf bytes.Buffer

	if err := bencode.NewMarshaller(&buf).Marshal(infoDict); err != nil {
		return [sha1.Size]byte{}, err
	}

	return sha1.Sum(buf.Bytes()), nil
}
