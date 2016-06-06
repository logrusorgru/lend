lend
====

[![GoDoc](https://godoc.org/github.com/logrusorgru/lend?status.svg)](https://godoc.org/github.com/logrusorgru/lend)
[![WTFPL License](https://img.shields.io/badge/license-wtfpl-blue.svg)](http://www.wtfpl.net/about/)
[![Build Status](https://travis-ci.org/logrusorgru/lend.svg)](https://travis-ci.org/logrusorgru/lend)
[![Coverage Status](https://coveralls.io/repos/logrusorgru/lend/badge.svg?branch=master)](https://coveralls.io/r/logrusorgru/lend?branch=master)
[![GoReportCard](http://goreportcard.com/badge/logrusorgru/lend)](http://goreportcard.com/report/logrusorgru/lend) | 
[![paypal donate](https://img.shields.io/badge/paypal%20%24-donate-orange.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=AVFWLEREA97PU)

Length delimited reader/writer/framing for TCP/UDP etc.

# Installation

Get
```
go get -u github.com/logrusorgru/lend
```
Test
go test github.com/logrusorgru/lend
```

# Usage

### From/to file

Write

```go
package main

import (
	"log"
	"os"

	"github.com/logrusorgru/lend"
)

var data = []string{
	"one",
	"two",
	"three",
}

const filename = "./data.bin"

func main() {
	fl, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fl.Close()
	buffered := bufio.NewWriter(fl)
	w, _ := lend.NewWriter(buffered, &lend.Config{Varint:true})
	for _, msg := range data {
		if err := w.Write([]byte(msg)); err != nil {
			log.Println("writing error:", err)
			return
		}
	}
	// flush the buffer
	if err := buffered.Flush(); err != nil {
		fmt.Println("flushing error:", err)
		return
	}
	log.Println("success")
}

```

Read

```go
package main

import (
	"log"
	"os"
	"io"

	"github.com/logrusorgru/lend"
)

const filename = "./data.bin"

func main() {
	fl, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fl.Close()
	// If Varint option is true then given io.Reader is
	// converted to io.ByteReader using bufio package.
	w, _ := lend.NewReader(fl, &lend.Config{Varint:true})
	for {
		if msg, err := w.Read(); err != nil {
			if err == io.EOF {
				break
			}
			log.Println("reading error:", err)
			return
		}
		fmt.Println(string(msg))
	}
	log.Println("success")
}

```

### TCP/UDP

See `examples_test.go` for TCP example. There is only one difference
between TCP and UDP networking. UDP can loose a data. This way you need
to enable framing delimiter.

```go
// create a Reader
r, _ := lend.NewReader(udpConnection, &Config{
	Heading: []byte("= SOME DELIMITER ="),
})
```

```go
// create a Writer
r, _ := lend.NewWriter(udpConnection, &Config{
	Heading: []byte("= SOME DELIMITER ="),
})
```

Keep in mind a Reader and Writer must have same configs (except Pool).
There is an option to use different MaxSize (size limit). But in all
cases this limit must be in the same range [1, max int32] or
(max int32, max int64]. If you enables varint encoding, then feel free
to choose this limit as you want (but greater that zero, actually).

Also, if Heading is a nil then all size limit errors will break a
Reader. It's possible to use some minimalistic Heading like
`[]byte{'H'}` to enable skipping all messages that  greater than
given limit.

### Pool

It's possible to provide your own pool. The Pool interface is

```go
Get(size int) []byte
Put([]byte)
```

Some example using `sync.Pool` where average size of messages
less than 100 bytes.

```go
const maxSize = 100

type pool struct {
	sync.Pool
}

func (p *pool) Put(piece []byte) {
	if cap(piece) > maxSize {
		p.Pool.Put(piece)
	}
	// drop large pieces
}

// Get must reurns slice with length
// equal to size (length not capacity!)
func (p *pool) Get(size int) []byte {
	if size > maxSize {
		return make([]byte, size)
	}
	if ifc := p.Pool.Get(); ifc != nil {
		return ifc.([]byte)[:size]
	}
	return make([]byte, size, maxSize)
}

// usage example

var p  = pool{} // create an instance of pool

func some() {
	r, _ = lend.NewReader(someReader, &lend.Config{
		Pool: &p, // provide the pool to this Reader
	})

	//
	// stuff
	//

}

```


### Licensing

Copyright &copy; 2015 Konstantin Ivanov <ivanov.konstantin@logrus.org.ru>  
This work is free. You can redistribute it and/or modify it under the
terms of the Do What The Fuck You Want To Public License, Version 2,
as published by Sam Hocevar. See the LICENSE.md file for more details.


