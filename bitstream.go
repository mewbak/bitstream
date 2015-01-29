// Package bitstream is used to read bits out of an io stream.
package bitstream

import (
	"bufio"
	"errors"
	"io"
)

var (
	bufferTooSmall = errors.New("bitsinbytes: buffer too small")
)

// Reader reads many different types of values outside byte alignments.
type Reader struct {
	reader *bufio.Reader

	offset uint
	bits   byte
}

func New(reader io.Reader) *Reader {
	return &Reader{
		offset: 8,
		reader: bufio.NewReader(reader),
	}
}

// Bits returns the next bits up to a max of 64.
func (r *Reader) Bits(nBits int) (val uint64, err error) {
	if nBits > 64 {
		panic("Can only read 64 bits at a time.")
	}

	var bitOffset uint
	for nBits > 0 {
		if r.offset == 8 {
			r.offset = 0
			r.bits, err = r.reader.ReadByte()
			if err != nil {
				return val, err
			}
		}

		toRead := uint(nBits)
		if toRead > (8 - r.offset) {
			toRead = 8 - r.offset
		}

		var mask byte = ((1 << toRead) - 1) << r.offset

		val |= (uint64(mask&r.bits) >> r.offset) << bitOffset
		bitOffset += toRead
		r.offset += toRead
		nBits -= int(toRead)
	}

	return val, nil
}

// Byte from the reader.
func (r *Reader) Byte() (byte, error) {
	if r.offset == 8 {
		return r.reader.ReadByte()
	}

	bits, err := r.Bits(8)
	return byte(bits), err
}

// Read whole bytes from the reader.
func (r *Reader) Read(dst []byte) (int, error) {
	if r.offset == 8 {
		ret, err := r.reader.Read(dst)

		// bufio doesn't fill it's buffer until it's completely empty.
		// if a short read happens with no error: retry.
		if err == nil && len(dst) != ret {
			again, e := r.reader.Read(dst[ret:])
			return again + ret, e
		}

		return ret, err
	}

	n := 0
	for i := 0; i < len(dst); i++ {
		bits, err := r.Bits(8)
		if err != nil {
			return n, err
		}

		dst[i] = byte(bits & 0xFF)
		n++
	}

	return n, nil
}

// Bytes returns the number of requested bits inside a byte array.
func (r *Reader) Bytes(dst []byte, nBits int) (err error) {
	var byteOffset int
	var bitOffset uint

	if len(dst) < (nBits+7)/8 {
		return bufferTooSmall
	}

	for nBits > 0 {
		if r.offset == 8 {
			r.offset = 0

			r.bits, err = r.reader.ReadByte()
			if err != nil {
				return err
			}
		}

		if bitOffset == 8 {
			bitOffset = 0
			byteOffset++
		}

		maskSize := uint(nBits)
		if maskSize > (8 - bitOffset) {
			maskSize = 8 - bitOffset
		}
		if maskSize > (8 - r.offset) {
			maskSize = 8 - r.offset
		}

		var mask byte = ((1 << maskSize) - 1) << r.offset

		dst[byteOffset] |= ((mask & r.bits) >> r.offset) << bitOffset
		bitOffset += maskSize
		r.offset += maskSize
		nBits -= int(maskSize)
	}

	return nil
}
