package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"sync"
)

// Block represents a blockh within a piece
type Block struct {
	Index  int    // Block index within the piece
	Begin  int    // Offset within the piece
	Length int    // Length of the block
	Data   []byte // Block data (nil if not downloaded)
}

// PieceState represents the state of a piece
type PieceState int

// Piece represents a piece of the torrent
type Piece struct {
	sync.RWMutex
	Index      int             // Piece index
	Length     int             // Length of the piece in bytes
	Downloaded int             // Number of bytes downloaded
	Blocks     []*Block        // Blocks within the piece
	State      PieceState      // Current state of the piece
	Requested  map[int]bool    // Tracks which blocks have been requested
	Hash       [sha1.Size]byte // Expected SHA1 hash
}

const (
	PieceStateNone PieceState = iota
	PieceStatePending
	PieceStateComplete
)

const BlockSize = 16 * 1024 // 16KB

func NewPiece(index, length int, hash [sha1.Size]byte) *Piece {
	numBlocks := length / BlockSize
	if length%BlockSize != 0 {
		numBlocks++
	}

	blocks := make([]*Block, numBlocks)

	for i := 0; i < numBlocks; i++ {
		begin := i * BlockSize
		blockLen := BlockSize

		if i == numBlocks-1 && length%BlockSize != 0 {
			blockLen = length % BlockSize
		}

		blocks[i] = &Block{Index: i, Begin: begin, Length: blockLen}
	}

	return &Piece{
		Index:     index,
		Hash:      hash,
		Length:    length,
		Blocks:    blocks,
		State:     PieceStateNone,
		Requested: make(map[int]bool),
	}
}

// MarkRequested marks a block as requested
func (p *Piece) MarkRequested(blockIndex int) {
	p.Lock()
	defer p.Unlock()

	if blockIndex < 0 || blockIndex > len(p.Blocks) {
		return
	}

	p.Requested[blockIndex] = true
	p.State = PieceStatePending
}

// AddBlock adds a downloaded block to the piece
func (p *Piece) AddBlock(begin int, data []byte) error {
	p.Lock()
	defer p.Unlock()

	for i, block := range p.Blocks {
		if begin != block.Begin {
			continue
		}

		if len(data) != block.Length {
			return fmt.Errorf(
				"block length mismatch: got %d, expected: %d",
				len(data),
				block.Length,
			)
		}

		p.Blocks[i].Data = data
		p.Downloaded += len(data)

		return nil
	}

	return fmt.Errorf("no block found with begin offset %d", begin)
}

// IsComplete returns true if all blocks have been downloaded
func (p *Piece) IsComplete() bool {
	p.RLock()
	defer p.RUnlock()

	return p.Length == p.Downloaded
}

// AssembleData copies all block data into a single byte slice
func (p *Piece) AssembleData() []byte {
	p.RLock()
	defer p.RUnlock()

	if !p.IsComplete() {
		return nil
	}

	data := make([]byte, p.Length)
	for _, block := range p.Blocks {
		if block.Data == nil {
			continue
		}

		copy(data[block.Begin:], block.Data)
	}

	return data
}

// Verify validates the piece integrity using SHA1 hash
func (p *Piece) Verify() bool {
	p.RLock()
	defer p.RUnlock()

	if !p.IsComplete() {
		return false
	}

	data := p.AssembleData()
	if data == nil {
		return false
	}

	hash := sha1.Sum(data)
	return bytes.Equal(p.Hash[:], hash[:])
}

// NextRequest requests next block to download, or nil if all blocks are requested
func (p *Piece) NextRequest() *Block {
	p.Lock()
	defer p.Unlock()

	for _, block := range p.Blocks {
		if block.Data == nil || p.Requested[block.Index] {
			continue
		}

		p.Requested[block.Index] = true
		return block
	}

	return nil
}

// GetState returns the state of the piece
func (p *Piece) GetState() PieceState {
	p.RLock()
	defer p.RUnlock()

	return p.State
}

// ResetRequests marks all blocks as not requested
func (p *Piece) ResetRequests() {
	p.Lock()
	defer p.Unlock()

	p.Requested = make(map[int]bool)
	if p.State == PieceStatePending {
		p.State = PieceStateNone
	}
}
