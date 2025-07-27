package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"

	"github.com/prxssh/relay/internal/bencode"
)

// Torrent represents the complete data from a .torrent file
type Torrent struct {
	// Announce URLs of the tracker. It combines both announce and announce-list.
	AnnounceURLs []string
	// Creation time of the torrent in UNIX epoch format (optional)
	CreationDate int64
	// Comments of the author (optional)
	Comment string
	// Name and version of the program used to create .torrent (optional)
	CreatedBy string
	// String encoding format used to generate the pieces part (optional)
	Encoding string
	// Describes the files of the torrent
	Info *Info
	// Size of this torrent
	Size int64
}

// Info contains the file-specific information of the torrent.
// This is the part of the metainfo that gets hashed to create the InfoHash
type Info struct {
	// Filename
	Name string
	// Number of bytes in each piece
	PieceLen int64
	// All the SHA1 hash of the pieces
	Pieces [][sha1.Size]byte
	// If true, client MUST publish its presence to get other peers ONLY via
	// the trackers explicitly described in the metainfo file.
	IsPrivate bool
	// Length of the file in bytes
	Length int64
	// Only present in multi-file mode
	Files []*File
	// SHA1 of the raw info dictionary
	Hash [sha1.Size]byte
}

// File represents a single file within a multi-file torrent
type File struct {
	// Length of file in bytes
	Length int64
	// MD5 sum of the file (optional)
	MD5 string
	// List containing one or more string elements that together represents the
	// path and filename.
	Path []string
}

func (m *Torrent) NumPieces() int {
	return len(m.Info.Pieces)
}

func New(r io.Reader) (*Torrent, error) {
	p, err := newParser(r)
	if err != nil {
		return nil, err
	}
	return p.parse()
}

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

/////////////// Private ///////////////

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
		return nil, errors.New(
			"metainfo: top-level is not a dictionary",
		)
	}

	return &parser{data: data}, nil
}

func (p *parser) parse() (*Torrent, error) {
	info, err := p.parseInfo()
	if err != nil {
		return nil, fmt.Errorf(
			"metainfo: failed to parse info dict: %w",
			err,
		)
	}

	announceURLs, err := p.parseAnnounce()
	if err != nil {
		return nil, err
	}

	return &Torrent{
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
		return nil, errors.New(
			"'info' key is missing or not a dictionary",
		)
	}

	infoHash, err := calculateSHA1Hash(infoDict)
	if err != nil {
		return nil, err
	}

	infoParser := &parser{data: infoDict}

	piecesStr, ok := infoParser.data["pieces"].(string)
	if !ok {
		return nil, errors.New(
			"'pieces' key is missing or not a string",
		)
	}
	if len(piecesStr)%sha1.Size != 0 {
		return nil, fmt.Errorf(
			"invalid pieces length %d",
			len(piecesStr),
		)
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
			pathStr, ok := pth.(string)
			if !ok {
				return nil, errors.New("file 'path' contains non-string element")
			}
			path[i] = pathStr

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

	if announce := p.getString("announce"); announce != "" {
		urls[announce] = struct{}{}
	}

	if len(urls) == 0 {
		return nil, errors.New("no trackers found in announce or announce-list")
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
