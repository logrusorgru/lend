//
// Copyright (c) 2016 Konstanin Ivanov <kostyarin.ivanov@gmail.com>.
// All rights reserved. This program is free software. It comes without
// any warranty, to the extent permitted by applicable law. You can
// redistribute it and/or modify it under the terms of the Do What
// The Fuck You Want To Public License, Version 2, as published by
// Sam Hocevar. See LICENSE.md file for more details or see below.
//

//
//        DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
//                    Version 2, December 2004
//
// Copyright (C) 2004 Sam Hocevar <sam@hocevar.net>
//
// Everyone is permitted to copy and distribute verbatim or modified
// copies of this license document, and changing it is allowed as long
// as the name is changed.
//
//            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
//   TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION
//
//  0. You just DO WHAT THE FUCK YOU WANT TO.
//

// Package lend implements length-delimited reader and writer. It also
// includes some framing mechanism for UDP network. It's possible to
// use a fixed-size length and a varint encoded length. The lend allows
// to use very large pieces of data, but if length of a piece is greater
// than max positive int32, then it's impossible to read/write it on 32-bit
// platforms. The framing mechanism uses user-provided delimiter that
// points to start of a piece of data. It's possible to limit max size
// of pieces. But if you uses a fixed-size length then for limit less
// than max positive int32, 4 bytes is used to keep the length. If limit
// is greater, then 8 bytes is used. The package is smiple and well-tested.
package lend

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

// A Pool represents a pool interface. There are not
// any internal pool. By default a Reader and a Writer
// uses make([]byte, ...) to create and nothing (GC) to
// drop slices. If your project can cause a lot of
// pressure on GC, then you can provide your own pool
// (sync.Pool wrapper or another) that will be used
// for creating/dropping slices.
type Pool interface {
	Get(size int) []byte
	Put([]byte)
}

// A Reader represents an interface that reads
// a data, piece by piece. Both, Reader and Writer
// should have the same configs MaxSize,
// Heading and Varint.
type Reader interface {
	Read() (piece []byte, err error)
}

// A Writer represents an interface that
// writes a data, piece by piece. Don't be
// entirely mislead. The piece argument
// below must be an entire piece. The
// Writer writes it, a Reader reads
// this piece.
type Writer interface {
	Write(piece []byte) (err error)
}

const (
	maxInt32 = int(^uint32(0) >> 1)
	maxInt   = int(^uint(0) >> 1)
)

type base struct {
	max     int
	pool    Pool
	varint  bool
	heading []byte
	lenb    []byte // used for reading length (avoid allocs)
}

type reader struct {
	r io.Reader
	b io.ByteReader
	base
	headbuf []byte // buffer that used to find heading
}

// A Config is a Reader and Writer configurations.
// See DefaultConfig() for defaults.
type Config struct {
	// MaxSize limits max piece size. If a piece
	// exceeds this limit then ErrSizeLimit will
	// be returned. By default it's max int32.
	// The maximum possible value is max int64.
	// But it's impossible to use values
	// greater that max positive int. So,
	// you can use max int64 only on x64
	// platforms.
	MaxSize int
	// Pool allows to provide own poll for
	// creating/dropping slices. By default
	// it's nil and make([]byte, ..) and GC
	// are used.
	Pool Pool
	// Heading is a framing options. It's used as
	// delimiter between pieces. By default
	// it's nil. If it's nil then no framing is
	// used and data stream is just length
	// delimited stream. But if a length of this
	// is greater than 0, then framing mechanism
	// is enabled. And data stream is Heading+length
	// delimited. This option makes sence only for
	// UDP and other protocols that can loose data.
	// This way, a Reader firstly finds a Heading.
	// Also, this way, the ErrSizeLimit and
	// ErrNegativeLength will never have.
	// All this errors will be treated as
	// wrong position and a frame will be skipped
	Heading []byte
	// Varint enables Varint encoding. By default,
	// if MaxSize is <= max int32 then a 4 bytes
	// are used to keep a length of a piece. If
	// MaxSize is greater than max int32 then
	// 8 bytes are used. This options allows to use
	// Varint encoding. This way, an io.Reader will
	// be converted to io.ByteReader. If it's not a
	// io.ByteReader then "bufio" package will be
	// used.
	Varint bool
}

// DefaultConfig returns default configurations.
// This used if *Config provided to a NewReader
// or a NewWriter is nil. By default MaxSize is
// max int32, Pool is nil, Heading is nil and
// Varint is false.
func DefaultConfig() *Config {
	return &Config{
		MaxSize: maxInt32,
	}
}

// Check validates configurations.
func (c *Config) Check() (err error) {
	if c.MaxSize <= 0 {
		err = errors.New("(*Config).MaxSize is negative or zero")
	}
	return
}

// NewReader creates Reader interface over given
// io.Reader using given *Config. If *Config
// is nil then DefaultConfig() is used. If given
// io.Reader is nil then first Read causes panic.
// Error indicates that *Config is incorrect
func NewReader(r io.Reader, c *Config) (Reader, error) {
	if c == nil {
		c = DefaultConfig()
	}
	if err := c.Check(); err != nil {
		return nil, err
	}
	q := new(reader)
	q.r = r
	q.max = int(c.MaxSize)
	q.pool = c.Pool
	q.varint = c.Varint
	q.heading = c.Heading
	if q.varint {
		q.makeByteReader()
	} else if c.MaxSize > maxInt32 {
		q.lenb = make([]byte, 8)
	} else {
		q.lenb = make([]byte, 4)
	}
	if len(q.heading) > 0 {
		q.headbuf = make([]byte, len(q.heading))
	}
	return q, nil
}

func (r *reader) makeByteReader() {
	if br, ok := r.r.(io.ByteReader); ok {
		r.b = br
		return
	}
	br := bufio.NewReader(r.r)
	r.b = br
	r.r = br
}

func (r *reader) get(size int) []byte {
	if r.pool != nil {
		return r.pool.Get(size)
	}
	return make([]byte, size)
}

// [asdf]
// [asaf]
//  01 - reset
//    2 - reset
//
// [aasd]
//  0 - reset
//   123 - got it

// x = 0
// n = x
// find n symbol from heading into hb, as m
// n++
// find n symbol from heading into hb
//  - got it: retry
//  - not found, x++, retry all
// finally
// m is a position of first symbol (from heading) that
// starts the sequance.
//   - move it to beginning
//   - read remainder
// retry all (inclusive cmp)

// len(hs) == len(ts)
func look(hs, ts []byte) (n int) {
startPos:
	for ; n < len(hs); n++ {
		var i, j int = n, 0
		for ; i < len(hs); i++ {
			if hs[j] == ts[i] {
				j++
				continue
			}
			continue startPos
		}
		return
	}
	return // call look(nil, nil) for 100% coverage
}

func (r *reader) findHeading() (err error) {
	for {
		var n int
		if _, err = io.ReadFull(r.r, r.headbuf); err != nil {
			return
		}
	lookAgain:
		if n = look(r.heading, r.headbuf); n == 0 { // got it!
			return
		}
		if n == len(r.heading) { // not found
			continue
		}
		// partially read
		copy(r.headbuf, r.headbuf[n:])
		if _, err = io.ReadFull(r.r, r.headbuf[len(r.headbuf)-n:]); err != nil {
			return
		}
		goto lookAgain
	}
}

var (
	// ErrNegativeLength occurs when a length value is negative.
	ErrNegativeLength = errors.New("negative length")
	// ErrSizeLimit means a length of a piece of data exceeds MaxSize option.
	ErrSizeLimit = errors.New("size limit exceeded")
)

// validate length
func (r *reader) validateLen(l int) error {
	if l < 0 {
		return ErrNegativeLength
	}
	if l > r.max {
		return ErrSizeLimit
	}
	return nil
}

func (r *reader) validateLen64(l64 int64) (l int, err error) {
	if l64 < 0 {
		err = ErrNegativeLength
		return
	}
	if l64 > int64(r.max) {
		err = ErrSizeLimit
		return
	}
	l = int(l64)
	return
}

func (r *reader) readLen() (l int, err error) {
	if r.varint {
		var l64 int64
		if l64, err = binary.ReadVarint(r.b); err != nil {
			return
		}
		return r.validateLen64(l64)
	}
	// read fixed size length
	if _, err = io.ReadFull(r.r, r.lenb); err != nil {
		return
	}
	if r.max <= maxInt32 {
		l = int(binary.BigEndian.Uint32(r.lenb))
		err = r.validateLen(l) // uint32
		return
	}
	return r.validateLen64(int64(binary.BigEndian.Uint64(r.lenb)))
}

func (r *reader) read() (piece []byte, err error) {
	var l int
	if l, err = r.readLen(); err != nil {
		return
	}
	piece = r.get(l)
	_, err = io.ReadFull(r.r, piece)
	return
}

func (r *reader) readWithHeading() (piece []byte, err error) {
retry:
	if err = r.findHeading(); err != nil {
		return
	}
	if piece, err = r.read(); err != nil {
		// not a reader error
		switch err {
		case ErrSizeLimit, ErrNegativeLength:
			goto retry
		}
	}
	return
}

// Read reads next piece of data.
func (r *reader) Read() ([]byte, error) {
	if len(r.heading) > 0 {
		return r.readWithHeading()
	}
	return r.read()
}

type writer struct {
	w io.Writer
	base
}

// NewWriter creates Writer interface over given
// io.Writer using given *Config. If *Config
// is nil then DefaultConfig() is used. If given
// io.Writer is nil then first Write causes panic.
// Error indicates that *Config is incorrect.
// If a Pool is given then each Write automatically
// puts a piece of data to the Pool. But if any
// error occurs during writing then the piece
// will not be put to the Pool.
func NewWriter(w io.Writer, c *Config) (_ Writer, err error) {
	if c == nil {
		c = DefaultConfig()
	}
	if err = c.Check(); err != nil {
		return
	}
	q := new(writer)
	q.w = w
	q.max = c.MaxSize
	q.pool = c.Pool
	q.varint = c.Varint
	q.heading = c.Heading
	if q.varint {
		q.lenb = make([]byte, 10) // for varints
	} else if q.max <= maxInt32 {
		q.lenb = make([]byte, 4)
	} else {
		q.lenb = make([]byte, 8)
	}
	return q, nil
}

func (w *writer) put(piece []byte) {
	if w.pool != nil {
		w.pool.Put(piece)
	}
}

// Write writes given piece to undelying io.Writer.
// It also writes nil and pieces wich lenength is 0.
// If a length of a piece exceeds a size limit
// then ErrSizeLimit is returned.
func (w *writer) Write(piece []byte) (err error) {
	if len(piece) > w.max {
		err = ErrSizeLimit
		return
	}
	if len(w.heading) > 0 {
		if _, err = w.w.Write(w.heading); err != nil {
			return
		}
	}
	if w.varint {
		n := binary.PutVarint(w.lenb, int64(len(piece)))
		if _, err = w.w.Write(w.lenb[:n]); err != nil {
			return
		}
	} else {
		if w.max <= maxInt32 {
			binary.BigEndian.PutUint32(w.lenb, uint32(len(piece)))
		} else {
			binary.BigEndian.PutUint64(w.lenb, uint64(len(piece)))
		}
		if _, err = w.w.Write(w.lenb); err != nil {
			return
		}
	}
	if _, err = w.w.Write(piece); err != nil {
		return
	}
	w.put(piece)
	return
}
