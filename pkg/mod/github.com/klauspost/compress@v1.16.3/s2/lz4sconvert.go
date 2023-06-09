// Copyright (c) 2022 Klaus Post. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s2

import (
	"encoding/binary"
	"fmt"
)

// LZ4sConverter provides conversion from LZ4s.
// (Intel modified LZ4 Blocks)
// https://cdrdv2-public.intel.com/743912/743912-qat-programmers-guide-v2.0.pdf
// LZ4s is a variant of LZ4 block format. LZ4s should be considered as an intermediate compressed block format.
// The LZ4s format is selected when the application sets the compType to CPA_DC_LZ4S in CpaDcSessionSetupData.
// The LZ4s block returned by the Intel® QAT hardware can be used by an external
// software post-processing to generate other compressed data formats.
// The following table lists the differences between LZ4 and LZ4s block format. LZ4s block format uses
// the same high-level formatting as LZ4 block format with the following encoding changes:
// For Min Match of 4 bytes, Copy length value 1-15 means length 4-18 with 18 bytes adding an extra byte.
// ONLY "Min match of 4 bytes" is supported.
type LZ4sConverter struct {
}

// ConvertBlock will convert an LZ4s block and append it as an S2
// block without block length to dst.
// The uncompressed size is returned as well.
// dst must have capacity to contain the entire compressed block.
func (l *LZ4sConverter) ConvertBlock(dst, src []byte) ([]byte, int, error) {
	if len(src) == 0 {
		return dst, 0, nil
	}
	const debug = false
	const inline = true
	const lz4MinMatch = 3

	s, d := 0, len(dst)
	dst = dst[:cap(dst)]
	if !debug && hasAmd64Asm {
		res, sz := cvtLZ4sBlockAsm(dst[d:], src)
		if res < 0 {
			const (
				errCorrupt     = -1
				errDstTooSmall = -2
			)
			switch res {
			case errCorrupt:
				return nil, 0, ErrCorrupt
			case errDstTooSmall:
				return nil, 0, ErrDstTooSmall
			default:
				return nil, 0, fmt.Errorf("unexpected result: %d", res)
			}
		}
		if d+sz > len(dst) {
			return nil, 0, ErrDstTooSmall
		}
		return dst[:d+sz], res, nil
	}

	dLimit := len(dst) - 10
	var lastOffset uint16
	var uncompressed int
	if debug {
		fmt.Printf("convert block start: len(src): %d, len(dst):%d \n", len(src), len(dst))
	}

	for {
		if s >= len(src) {
			return dst[:d], 0, ErrCorrupt
		}
		// Read literal info
		token := src[s]
		ll := int(token >> 4)
		ml := int(lz4MinMatch + (token & 0xf))

		// If upper nibble is 15, literal length is extended
		if token >= 0xf0 {
			for {
				s++
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ll: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return dst[:d], 0, ErrCorrupt
				}
				val := src[s]
				ll += int(val)
				if val != 255 {
					break
				}
			}
		}
		// Skip past token
		if s+ll >= len(src) {
			if debug {
				fmt.Printf("error literals: s+ll (%d+%d) >= len(src) (%d)\n", s, ll, len(src))
			}
			return nil, 0, ErrCorrupt
		}
		s++
		if ll > 0 {
			if d+ll > dLimit {
				return nil, 0, ErrDstTooSmall
			}
			if debug {
				fmt.Printf("emit %d literals\n", ll)
			}
			d += emitLiteralGo(dst[d:], src[s:s+ll])
			s += ll
			uncompressed += ll
		}

		// Check if we are done...
		if ml == lz4MinMatch {
			if s == len(src) {
				break
			}
			// 0 bytes.
			continue
		}
		// 2 byte offset
		if s >= len(src)-2 {
			if debug {
				fmt.Printf("s (%d) >= len(src)-2 (%d)", s, len(src)-2)
			}
			return nil, 0, ErrCorrupt
		}
		offset := binary.LittleEndian.Uint16(src[s:])
		s += 2
		if offset == 0 {
			if debug {
				fmt.Printf("error: offset 0, ml: %d, len(src)-s: %d\n", ml, len(src)-s)
			}
			return nil, 0, ErrCorrupt
		}
		if int(offset) > uncompressed {
			if debug {
				fmt.Printf("error: offset (%d)> uncompressed (%d)\n", offset, uncompressed)
			}
			return nil, 0, ErrCorrupt
		}

		if ml == lz4MinMatch+15 {
			for {
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return nil, 0, ErrCorrupt
				}
				val := src[s]
				s++
				ml += int(val)
				if val != 255 {
					if s >= len(src) {
						if debug {
							fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
						}
						return nil, 0, ErrCorrupt
					}
					break
				}
			}
		}
		if offset == lastOffset {
			if debug {
				fmt.Printf("emit repeat, length: %d, offset: %d\n", ml, offset)
			}
			if !inline {
				d += emitRepeat16(dst[d:], offset, ml)
			} else {
				length := ml
				dst := dst[d:]
				for len(dst) > 5 {
					// Repeat offset, make length cheaper
					length -= 4
					if length <= 4 {
						dst[0] = uint8(length)<<2 | tagCopy1
						dst[1] = 0
						d += 2
						break
					}
					if length < 8 && offset < 2048 {
						// Encode WITH offset
						dst[1] = uint8(offset)
						dst[0] = uint8(offset>>8)<<5 | uint8(length)<<2 | tagCopy1
						d += 2
						break
					}
					if length < (1<<8)+4 {
						length -= 4
						dst[2] = uint8(length)
						dst[1] = 0
						dst[0] = 5<<2 | tagCopy1
						d += 3
						break
					}
					if length < (1<<16)+(1<<8) {
						length -= 1 << 8
						dst[3] = uint8(length >> 8)
						dst[2] = uint8(length >> 0)
						dst[1] = 0
						dst[0] = 6<<2 | tagCopy1
						d += 4
						break
					}
					const maxRepeat = (1 << 24) - 1
					length -= 1 << 16
					left := 0
					if length > maxRepeat {
						left = length - maxRepeat + 4
						length = maxRepeat - 4
					}
					dst[4] = uint8(length >> 16)
					dst[3] = uint8(length >> 8)
					dst[2] = uint8(length >> 0)
					dst[1] = 0
					dst[0] = 7<<2 | tagCopy1
					if left > 0 {
						d += 5 + emitRepeat16(dst[5:], offset, left)
						break
					}
					d += 5
					break
				}
			}
		} else {
			if debug {
				fmt.Printf("emit copy, length: %d, offset: %d\n", ml, offset)
			}
			if !inline {
				d += emitCopy16(dst[d:], offset, ml)
			} else {
				length := ml
				dst := dst[d:]
				for len(dst) > 5 {
					// Offset no more than 2 bytes.
					if length > 64 {
						off := 3
						if offset < 2048 {
							// emit 8 bytes as tagCopy1, rest as repeats.
							dst[1] = uint8(offset)
							dst[0] = uint8(offset>>8)<<5 | uint8(8-4)<<2 | tagCopy1
							length -= 8
							off = 2
						} else {
							// Emit a length 60 copy, encoded as 3 bytes.
							// Emit remaining as repeat value (minimum 4 bytes).
							dst[2] = uint8(offset >> 8)
							dst[1] = uint8(offset)
							dst[0] = 59<<2 | tagCopy2
							length -= 60
						}
						// Emit remaining as repeats, at least 4 bytes remain.
						d += off + emitRepeat16(dst[off:], offset, length)
						break
					}
					if length >= 12 || offset >= 2048 {
						// Emit the remaining copy, encoded as 3 bytes.
						dst[2] = uint8(offset >> 8)
						dst[1] = uint8(offset)
						dst[0] = uint8(length-1)<<2 | tagCopy2
						d += 3
						break
					}
					// Emit the remaining copy, encoded as 2 bytes.
					dst[1] = uint8(offset)
					dst[0] = uint8(offset>>8)<<5 | uint8(length-4)<<2 | tagCopy1
					d += 2
					break
				}
			}
			lastOffset = offset
		}
		uncompressed += ml
		if d > dLimit {
			return nil, 0, ErrDstTooSmall
		}
	}

	return dst[:d], uncompressed, nil
}

// ConvertBlockSnappy will convert an LZ4s block and append it
// as a Snappy block without block length to dst.
// The uncompressed size is returned as well.
// dst must have capacity to contain the entire compressed block.
func (l *LZ4sConverter) ConvertBlockSnappy(dst, src []byte) ([]byte, int, error) {
	if len(src) == 0 {
		return dst, 0, nil
	}
	const debug = false
	const lz4MinMatch = 3

	s, d := 0, len(dst)
	dst = dst[:cap(dst)]
	// Use assembly when possible
	if !debug && hasAmd64Asm {
		res, sz := cvtLZ4sBlockSnappyAsm(dst[d:], src)
		if res < 0 {
			const (
				errCorrupt     = -1
				errDstTooSmall = -2
			)
			switch res {
			case errCorrupt:
				return nil, 0, ErrCorrupt
			case errDstTooSmall:
				return nil, 0, ErrDstTooSmall
			default:
				return nil, 0, fmt.Errorf("unexpected result: %d", res)
			}
		}
		if d+sz > len(dst) {
			return nil, 0, ErrDstTooSmall
		}
		return dst[:d+sz], res, nil
	}

	dLimit := len(dst) - 10
	var uncompressed int
	if debug {
		fmt.Printf("convert block start: len(src): %d, len(dst):%d \n", len(src), len(dst))
	}

	for {
		if s >= len(src) {
			return nil, 0, ErrCorrupt
		}
		// Read literal info
		token := src[s]
		ll := int(token >> 4)
		ml := int(lz4MinMatch + (token & 0xf))

		// If upper nibble is 15, literal length is extended
		if token >= 0xf0 {
			for {
				s++
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ll: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return nil, 0, ErrCorrupt
				}
				val := src[s]
				ll += int(val)
				if val != 255 {
					break
				}
			}
		}
		// Skip past token
		if s+ll >= len(src) {
			if debug {
				fmt.Printf("error literals: s+ll (%d+%d) >= len(src) (%d)\n", s, ll, len(src))
			}
			return nil, 0, ErrCorrupt
		}
		s++
		if ll > 0 {
			if d+ll > dLimit {
				return nil, 0, ErrDstTooSmall
			}
			if debug {
				fmt.Printf("emit %d literals\n", ll)
			}
			d += emitLiteralGo(dst[d:], src[s:s+ll])
			s += ll
			uncompressed += ll
		}

		// Check if we are done...
		if ml == lz4MinMatch {
			if s == len(src) {
				break
			}
			// 0 bytes.
			continue
		}
		// 2 byte offset
		if s >= len(src)-2 {
			if debug {
				fmt.Printf("s (%d) >= len(src)-2 (%d)", s, len(src)-2)
			}
			return nil, 0, ErrCorrupt
		}
		offset := binary.LittleEndian.Uint16(src[s:])
		s += 2
		if offset == 0 {
			if debug {
				fmt.Printf("error: offset 0, ml: %d, len(src)-s: %d\n", ml, len(src)-s)
			}
			return nil, 0, ErrCorrupt
		}
		if int(offset) > uncompressed {
			if debug {
				fmt.Printf("error: offset (%d)> uncompressed (%d)\n", offset, uncompressed)
			}
			return nil, 0, ErrCorrupt
		}

		if ml == lz4MinMatch+15 {
			for {
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return nil, 0, ErrCorrupt
				}
				val := src[s]
				s++
				ml += int(val)
				if val != 255 {
					if s >= len(src) {
						if debug {
							fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
						}
						return nil, 0, ErrCorrupt
					}
					break
				}
			}
		}
		if debug {
			fmt.Printf("emit copy, length: %d, offset: %d\n", ml, offset)
		}
		length := ml
		// d += emitCopyNoRepeat(dst[d:], int(offset), ml)
		for length > 0 {
			if d >= dLimit {
				return nil, 0, ErrDstTooSmall
			}

			// Offset no more than 2 bytes.
			if length > 64 {
				// Emit a length 64 copy, encoded as 3 bytes.
				dst[d+2] = uint8(offset >> 8)
				dst[d+1] = uint8(offset)
				dst[d+0] = 63<<2 | tagCopy2
				length -= 64
				d += 3
				continue
			}
			if length >= 12 || offset >= 2048 || length < 4 {
				// Emit the remaining copy, encoded as 3 bytes.
				dst[d+2] = uint8(offset >> 8)
				dst[d+1] = uint8(offset)
				dst[d+0] = uint8(length-1)<<2 | tagCopy2
				d += 3
				break
			}
			// Emit the remaining copy, encoded as 2 bytes.
			dst[d+1] = uint8(offset)
			dst[d+0] = uint8(offset>>8)<<5 | uint8(length-4)<<2 | tagCopy1
			d += 2
			break
		}
		uncompressed += ml
		if d > dLimit {
			return nil, 0, ErrDstTooSmall
		}
	}

	return dst[:d], uncompressed, nil
}
