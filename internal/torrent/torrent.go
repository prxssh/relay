package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"

	"github.com/prxssh/relay/internal/bencode"
	"github.com/prxssh/relay/internal/utils"
)

type File struct {
	Length int64
	MD5    string
	Path   []string
}

type Info struct {
	Name     string
	Length   int64
	Files    []*File
	PieceLen int64
	Pieces   [][sha1.Size]byte
	Private  bool
}

type Torrent struct {
	AnnounceList []string
	CreationDate int64
	Comment      string
	CreatedBy    string
	Encoding     string
	Info         *Info
	Size         int64
	InfoHash     [sha1.Size]byte
}

func UnmarshalMetainfo(r io.Reader) (*Torrent, error) {
	unmarshalled, err := bencode.NewUnmarshaller(r).Unmarshal()
	if err != nil {
		return nil, err
	}

	meta, ok := unmarshalled.(map[string]any)
	if !ok {
		return nil, errors.New("torrent decode: top-level is not a dictionary")
	}

	announceURL, err := utils.MapGetString(meta, "announce", true)
	if err != nil {
		return nil, err
	}

	announceList, err := parseAnnounceList(meta)
	if err != nil {
		return nil, err
	}

	creationDate, err := utils.MapGetInt(meta, "creation date", false)
	if err != nil {
		return nil, err
	}

	createdBy, err := utils.MapGetString(meta, "created by", false)
	if err != nil {
		return nil, err
	}

	encoding, err := utils.MapGetString(meta, "encoding", false)
	if err != nil {
		return nil, err
	}

	comment, err := utils.MapGetString(meta, "comment", false)
	if err != nil {
		return nil, err
	}

	infoRaw, ok := meta["info"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("info key missing from .torrent")
	}

	info, err := parseInfoDict(infoRaw)
	if err != nil {
		return nil, err
	}

	infoHash, err := calculateSHA1Hash(infoRaw)
	if err != nil {
		return nil, err
	}

	return &Torrent{
		CreationDate: creationDate,
		CreatedBy:    createdBy,
		Encoding:     encoding,
		Comment:      comment,
		Info:         info,
		InfoHash:     infoHash,
		Size:         calculateTorrentSize(info),
		AnnounceList: append(announceList, announceURL),
	}, nil
}

func parseInfoDict(d map[string]any) (*Info, error) {
	pieceLen, err := utils.MapGetInt(d, "piece length", true)
	if err != nil {
		return nil, err
	}

	name, err := utils.MapGetString(d, "name", true)
	if err != nil {
		return nil, err
	}

	pstr, err := utils.MapGetString(d, "pieces", true)
	if err != nil {
		return nil, err
	}
	pbytes := []byte(pstr)
	plen := len(pbytes)
	if plen%20 != 0 {
		return nil, fmt.Errorf("invalid pieces length %d; must be multiple of 20", plen)
	}
	var pieces [][sha1.Size]byte
	for i := 0; i < plen; i += 20 {
		var p [sha1.Size]byte

		copy(p[:], pbytes[i:i+20])
		pieces = append(pieces, p)
	}

	length, err := utils.MapGetInt(d, "length", false)
	if err != nil {
		return nil, err
	}

	files, err := parseFiles(d)
	if err != nil {
		return nil, err
	}

	return &Info{
		Name:     name,
		Pieces:   pieces,
		Length:   length,
		Files:    files,
		PieceLen: pieceLen,
	}, nil
}

func parseAnnounceList(meta map[string]any) ([]string, error) {
	raw, ok := meta["announce-list"]
	if !ok {
		return []string{}, nil // optional
	}

	announceListRaw, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("announce-list is not a list")
	}

	var out []string
	for _, group := range announceListRaw {
		inner, ok := group.([]any)
		if !ok {
			return nil, fmt.Errorf("announce-list group is not a list")
		}

		for _, u := range inner {
			s, ok := u.(string)
			if !ok {
				return nil, fmt.Errorf("announce-list URL is not a string")
			}
			out = append(out, s)
		}
	}

	return out, nil
}

func parseFiles(m map[string]any) ([]*File, error) {
	rawFiles, ok := m["files"]
	if !ok {
		return nil, nil // optional
	}

	list, ok := rawFiles.([]any)
	if !ok {
		return nil, fmt.Errorf("'files' is not a list")
	}

	var files []*File
	for _, entry := range list {
		m, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("file entry is not a dict")
		}

		length, err := utils.MapGetInt(m, "length", true)
		if err != nil {
			return nil, err
		}

		md5sum, err := utils.MapGetString(m, "md5sum", false)
		if err != nil {
			return nil, err
		}

		rawPath, ok := m["path"].([]any)
		if !ok {
			return nil, fmt.Errorf("'path' is not alist")
		}
		path := make([]string, len(rawPath))
		for i, p := range rawPath {
			s, ok := p.(string)
			if !ok {
				return nil, fmt.Errorf("path element is not a string")
			}
			path[i] = s
		}

		files = append(
			files,
			&File{Length: length, Path: path, MD5: md5sum},
		)
	}

	return files, nil
}

func calculateSHA1Hash(infoDict map[string]any) ([sha1.Size]byte, error) {
	var buf bytes.Buffer

	if err := bencode.NewMarshaller(&buf).Marshal(infoDict); err != nil {
		return [sha1.Size]byte{}, err
	}

	return sha1.Sum(buf.Bytes()), nil
}

func calculateTorrentSize(info *Info) int64 {
	if len(info.Files) == 0 {
		return info.Length
	}

	var size int64
	for _, f := range info.Files {
		size += f.Length
	}

	return size
}
