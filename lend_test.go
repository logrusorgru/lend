//
// Copyright (c) 2016 Konstanin Ivanov <kostyarin.ivanov@gmail.com>.
// All rights reserved. This program is free software. It comes without
// any warranty, to the extent permitted by applicable law. You can
// redistribute it and/or modify it under the terms of the Do What
// The Fuck You Want To Public License, Version 2, as published by
// Sam Hocevar. See LICENSE file for more details or see below.
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

package lend

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

/*

go test -coverprofile cover.out && go tool cover -html=cover.out -o cover.html

*/

func TestDefaultConfig(t *testing.T) {
	d := DefaultConfig()
	if d == nil {
		t.Fatal("DefaultConfig() returns nil")
	}
}

func TestConfig_Check(t *testing.T) {
	if err := (&Config{MaxSize: -1}).Check(); err == nil {
		t.Error("(*Config).Check(): missing error on negative MaxSize")
	}
	if err := (&Config{MaxSize: 0}).Check(); err == nil {
		t.Error("(*Config).Check(): missing error on zero MaxSize")
	}
}

func Test_look(t *testing.T) {
	{ // equal
		hs := []byte("hello")
		ts := hs
		if n := look(hs, ts); n != 0 {
			t.Error("look(equal) wrong value; expected 0, got", n)
		}
	}
	{ // not equal
		hs := []byte("hello")
		ts := []byte("ppppp")
		if n := look(hs, ts); n != len(hs) {
			t.Errorf("look(equal) wrong value; expected %d, got %d", len(hs), n)
		}
	}
	{ // shifted
		hs := []byte("hello")
		ts := []byte("ppphe")
		if n := look(hs, ts); n != 3 {
			t.Error("look(equal) wrong value; expected 3, got", n)
		}
	}
	{ // shifted (first)
		hs := []byte("hhhhh")
		ts := []byte("phhhh")
		if n := look(hs, ts); n != 1 {
			t.Error("look(equal) wrong value; expected 1, got", n)
		}
	}
	{ // coverage
		if n := look(nil, nil); n != 0 {
			t.Error("look(nil, nil): expected 0, got", n)
		}
	}
}

func TestNewReader(t *testing.T) {
	r, err := NewReader(nil, nil)
	if err != nil {
		t.Fatal("NewReader returns an unexpected error:", err)
	}
	if r == nil {
		t.Fatal("NewReader return nil")
	}
}

func TestNewReader_badConfigs(t *testing.T) {
	if _, err := NewReader(nil, &Config{MaxSize: -1}); err == nil {
		t.Fatal("NewReader with bad configs: missing error")
	}
}

func TestNewReader_varintWithByteReader(t *testing.T) {
	und := new(bytes.Buffer)
	r, err := NewReader(und, &Config{MaxSize: maxInt, Varint: true})
	if err != nil {
		t.Fatal("NewReader unexpected error:", err)
	}
	if r.(*reader).b == nil {
		t.Error("NewReader doesn't create ByteReader")
	}
}

func TestNewReader_varintWithoutByteReader(t *testing.T) {
	r, err := NewReader(nil, &Config{MaxSize: maxInt, Varint: true})
	if err != nil {
		t.Fatal("NewReader unexpected error:", err)
	}
	if r.(*reader).b == nil {
		t.Error("NewReader doesn't create ByteReader")
	}
}

func TestNewReader_uint32(t *testing.T) {
	r, err := NewReader(nil, &Config{MaxSize: maxInt32})
	if err != nil {
		t.Fatal("NewReader unexpected error:", err)
	}
	if r.(*reader).b != nil {
		t.Error("NewReader creates ByteReader for fixed size encoding")
	}
	l := len(r.(*reader).lenb)
	if l != 4 {
		t.Error("NewReader unexpected length of length buffer:", l)
	}
}

func TestNewReader_uint64(t *testing.T) {
	if maxInt == maxInt32 {
		t.Skip("platform depended test requires 64-bit int size")
	}
	max := maxInt32
	buf := new(bytes.Buffer)
	r, err := NewReader(buf, &Config{MaxSize: max + 1})
	if err != nil {
		t.Fatal("NewReader unexpected error:", err)
	}
	if r.(*reader).b != nil {
		t.Error("NewReader creates ByteReader for fixed size encoding")
	}
	l := len(r.(*reader).lenb)
	if l != 8 {
		t.Error("NewReader unexpected length of length buffer:", l)
	}
	p := "piece"
	lenb := make([]byte, 8)
	binary.BigEndian.PutUint64(lenb, uint64(len(p)))
	buf.Write(lenb)
	buf.WriteString(p)
	if pc, err := r.Read(); err != nil {
		t.Error("unexpected error:", err)
	} else if string(pc) != p {
		t.Error("wrong data, want 'piece', got:", string(pc))
	}
}

func TestNewReader_heading(t *testing.T) {
	heading := []byte("heading")
	r, err := NewReader(nil, &Config{MaxSize: maxInt32, Heading: heading})
	if err != nil {
		t.Fatal("NewReader unexpected error:", err)
	}
	rr := r.(*reader)
	if string(rr.heading) != string(heading) {
		t.Error("NewReader doesn't keep heading")
	}
	if len(rr.headbuf) != len(heading) {
		t.Errorf("NewReader wrong headbuf len, want %d, got: %d:",
			len(rr.headbuf), len(heading))
	}
}

// func writeString(buf *bytes.Buffer, msg string) {
// 	lenb := make([]byte, 4)
// 	binary.BigEndian.PutUint32(lenb, uint32(len(msg)))
// 	buf.Write(lenb)
// 	buf.WriteString(msg)
// }

// very synthetic (for the great coverage!)
func TestReader_validateLen(t *testing.T) {
	r, err := NewReader(nil, &Config{MaxSize: 3})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	rr := r.(*reader)
	if err := rr.validateLen(-10); err == nil {
		t.Error("missing negative len err")
	}
	if err := rr.validateLen(4); err == nil {
		t.Error("missing size limit err")
	}
}

// very synthetic (for the great coverage!)
func TestReader_validateLen64(t *testing.T) {
	r, err := NewReader(nil, &Config{MaxSize: 3})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	rr := r.(*reader)
	if _, err := rr.validateLen64(-10); err == nil {
		t.Error("missing negative len err")
	}
	if _, err := rr.validateLen64(4); err == nil {
		t.Error("missing size limit err")
	}
	if l, err := rr.validateLen64(2); err != nil {
		t.Error("unexpected error:", err)
	} else if l != 2 {
		t.Error("wrong len after validation (64)")
	}
}

func Test_reader_writer_nil(t *testing.T) {
	buf := new(bytes.Buffer)
	w, err := NewWriter(buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := w.Write([]byte("Hello, Lend!")); err != nil {
			t.Error(err)
		}
		p, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		if string(p) != "Hello, Lend!" {
			t.Error("wrong value")
		}
	}
}

func Test_reader_writer_varint(t *testing.T) {
	c := &Config{MaxSize: 100, Varint: true}
	buf := new(bytes.Buffer)
	w, err := NewWriter(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := w.Write([]byte("Hello, Lend!")); err != nil {
			t.Error(err)
		}
		p, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		if string(p) != "Hello, Lend!" {
			t.Error("wrong value")
		}
	}
}

func Test_reader_writer_heading(t *testing.T) {
	c := &Config{MaxSize: 100, Heading: []byte("HEAD")}
	buf := new(bytes.Buffer)
	w, err := NewWriter(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := w.Write([]byte("Hello, Lend!")); err != nil {
			t.Error(err)
		}
		p, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		if string(p) != "Hello, Lend!" {
			t.Error("wrong value")
		}
	}
}

type dummyPool struct{}

func (d dummyPool) Get(size int) []byte { return make([]byte, size) }
func (d dummyPool) Put([]byte)          {}

func Test_reader_writer_pool(t *testing.T) {
	c := &Config{MaxSize: 100, Pool: dummyPool{}}
	buf := new(bytes.Buffer)
	w, err := NewWriter(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(buf, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := w.Write([]byte("Hello, Lend!")); err != nil {
			t.Error(err)
		}
		p, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		if string(p) != "Hello, Lend!" {
			t.Error("wrong value")
		}
	}
}

type errorReader struct{}

func (e errorReader) Read([]byte) (_ int, err error) {
	err = errors.New("some error")
	return
}

func Test_reader_errorReader_nil(t *testing.T) {
	r, err := NewReader(errorReader{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

func Test_reader_errorReader_varint(t *testing.T) {
	r, err := NewReader(errorReader{}, &Config{MaxSize: 10, Varint: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

func Test_reader_errorReader_heading(t *testing.T) {
	heading := []byte("HEAD")
	r, err := NewReader(errorReader{}, &Config{MaxSize: 10, Heading: heading})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

type errorAfterContent struct {
	c []byte
}

func (e *errorAfterContent) Read(p []byte) (n int, err error) {
	if len(e.c) == 0 {
		err = errors.New("an error")
	}
	n = copy(p, e.c)
	e.c = e.c[n:]
	return
}

func Test_errorAfterContent(t *testing.T) {
	c := []byte("HEAD")
	e := errorAfterContent{c: c}
	for _, b := range c {
		buf := []byte{0}
		if n, err := e.Read(buf); err != nil {
			t.Fatal(err)
		} else if n != 1 {
			t.Fatal("want 1, got:", n)
		}
		if buf[0] != b {
			t.Fatalf("want %q, got %q", string(b), string(buf[0]))
		}
	}
	buf := []byte{0}
	if _, err := e.Read(buf); err == nil {
		t.Error("missing error")
	}
}

// synthetic (for the great coverage!)
func Test_reader_heading_shifted_err(t *testing.T) {
	heading := []byte("HEAD")
	r, err := NewReader(
		&errorAfterContent{c: []byte("HEAHEAD")},
		&Config{MaxSize: 10, Heading: heading},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

// synthetic (for the great coverage!)
func Test_reader_heading_shifted_err2(t *testing.T) {
	heading := []byte("HEAD")
	r, err := NewReader(
		&errorAfterContent{c: []byte("HEAHEA")},
		&Config{MaxSize: 10, Heading: heading},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

// synthetic (for the great coverage!)
func Test_reader_heading_not_found(t *testing.T) {
	heading := []byte("HEAD")
	r, err := NewReader(
		&errorAfterContent{c: []byte("NNNNNNNNNN")},
		&Config{MaxSize: 10, Heading: heading},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Error("missing error")
	}
}

func writeVarint(buf *bytes.Buffer, l int) {
	b := make([]byte, 10)
	n := binary.PutVarint(b, int64(l))
	buf.Write(b[:n])
}

// synthetic (for the great coverage!)
func Test_reader_heading_errs_to_skip(t *testing.T) {
	buf := new(bytes.Buffer)
	heading := []byte("HEAD")
	//
	buf.Write(heading)
	writeVarint(buf, -1)
	//
	buf.Write(heading)
	writeVarint(buf, 10)
	//
	buf.Write(heading)
	writeVarint(buf, 3)
	buf.WriteString("asd")
	//
	buf.WriteString("omit EOF")
	r, err := NewReader(buf, &Config{
		MaxSize: 3,
		Heading: heading,
		Varint:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p, err := r.Read(); err != nil {
		t.Error("unexpected error:", err)
	} else if string(p) != "asd" {
		t.Errorf("wrong data, want %q, got %q", "asd", string(p))
	}
}

func TestNewWriter_badConfigs(t *testing.T) {
	if _, err := NewWriter(nil, &Config{MaxSize: -1}); err == nil {
		t.Fatal("NewWriter with bad configs: missing error")
	}
}

func Test_writer_big_piece(t *testing.T) {
	w, err := NewWriter(nil, &Config{MaxSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte("I'm a big piece")); err == nil {
		t.Error("missing error for big piece")
	}
}

type errorWriter struct{}

func (e errorWriter) Write([]byte) (_ int, err error) {
	err = errors.New("some error")
	return
}

// very synthetic (for the great coverage!)
func Test_writer_heading_err(t *testing.T) {
	w, err := NewWriter(errorWriter{}, &Config{
		MaxSize: 100, Heading: []byte("h"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte("I'm a piece")); err == nil {
		t.Error("missing error for writing heading")
	}
}

// very synthetic (for the great coverage!)
func Test_writer_varint_err(t *testing.T) {
	w, err := NewWriter(errorWriter{}, &Config{
		MaxSize: 100, Varint: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte("I'm a piece")); err == nil {
		t.Error("missing error for writing heading")
	}
}

// very synthetic (for the great coverage!)
func Test_writer_uint32_err(t *testing.T) {
	w, err := NewWriter(errorWriter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte("I'm a piece")); err == nil {
		t.Error("missing error for writing heading")
	}
}

type secondWriteErr bool

func (s *secondWriteErr) Write([]byte) (_ int, err error) {
	if *s {
		err = errors.New("some error")
		return
	}
	*s = true
	return
}

// very synthetic (for the great coverage!)
func Test_writer_uint32_second_err(t *testing.T) {
	var swe secondWriteErr
	w, err := NewWriter(&swe, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte("I'm a piece")); err == nil {
		t.Error("missing error for writing heading")
	}
}

func TestNewWriter_uint64(t *testing.T) {
	if maxInt == maxInt32 {
		t.Skip("platform depended test requires 64-bit int size")
	}
	max := maxInt32
	buf := new(bytes.Buffer)
	r, err := NewWriter(buf, &Config{MaxSize: max + 1})
	if err != nil {
		t.Fatal("NewWriter unexpected error:", err)
	}
	l := len(r.(*writer).lenb)
	if l != 8 {
		t.Error("NewWriter unexpected length of length buffer:", l)
	}
	if err := r.Write([]byte("piece")); err != nil {
		t.Error(err)
	} else if buf.Len() != 8+len("piece") {
		t.Error("wrong size written")
	}
}
