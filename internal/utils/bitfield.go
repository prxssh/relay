package utils

type Bitfield []byte

func NewBitfield(size int) Bitfield {
	szBytes := (size + 7) / 8
	return make(Bitfield, szBytes)
}

func (bf Bitfield) Has(index int) bool {
	byteIndex, bitIndex := index/8, index%8

	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}

	// We create a "mask" that has only one bit set at the position we care
	// about. The bits are typically ordered from most-significant (left) to
	// least-significant (right). So, for bitIndex 0, we need to check the
	// leftmost bit (10000000). For bitIndex 2, we need to check the third bit
	// from the left (00100000). The mask `1 << (7 - bitIndex)` achieves this.
	//
	// Example for index 10 (byteIndex=1, bitIndex=2):
	// Mask = 1 << (7 - 2) = 1 << 5 = 00100000 (binary)
	//
	// If the byte from the bitfield is `10110101`:
	//   10110101 (byte)
	// & 00100000 (mask)
	// ----------
	//   00100000 (result is not zero, so the bit was set)
	//
	// If the byte was `10010101`:
	//   10010101 (byte)
	// & 00100000 (mask)
	// ----------
	//   00000000 (result is zero, so the bit was not set)
	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}

func (bf Bitfield) Set(index int) {
	byteIndex, bitIndex := index/8, index%8

	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}

	// We use the same mask as before. The OR operation sets the bit at the mask's
	// position to 1 without affecting any other bits.
	//
	// Example for index 10 (byteIndex=1, bitIndex=2):
	// Mask = 1 << (7 - 2) = 00100000 (binary)
	//
	// If the byte is `10010101`:
	//   10010101 (byte)
	// | 00100000 (mask)
	// ----------
	//   10110101 (the new value of the byte)
	bf[byteIndex] |= (1 << (7 - bitIndex))
}
