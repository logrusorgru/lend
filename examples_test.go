package lend

import (
	"fmt"
	"log"
	"net"
	"sync"
)

func Example_tcp() {
	const (
		bind = "127.0.0.1:9100"
	)
	send := []string{
		"Hello",
		"How are you?",
		"Bye",
	}
	reply := []string{
		"Hi",
		"Fine",
		"Cya",
	}
	// Start TCP server
	server, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	// waiting for server using this group
	wg := new(sync.WaitGroup)
	wg.Add(1)
	// accept connection
	go func() {
		defer wg.Done()
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// error is referenced to configs, ignore it
		r, _ := NewReader(conn, nil)
		w, _ := NewWriter(conn, nil)
		// read and print incoming messages
		for i := 0; i < len(send); i++ {
			msg, err := r.Read()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("C:", string(msg))
			if err := w.Write([]byte(reply[i])); err != nil {
				log.Fatal(err)
			}
		}
		conn.Close()
		server.Close()
	}()
	// Create client
	client, err := net.Dial("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	// error is referenced to configs, ignore it
	w, _ := NewWriter(client, nil)
	r, _ := NewReader(client, nil)
	for _, msg := range send {
		if err := w.Write([]byte(msg)); err != nil {
			log.Fatal(err)
		}
		rep, err := r.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("S:", string(rep))
	}
	client.Close()
	wg.Wait()
	// Output:
	// C: Hello
	// S: Hi
	// C: How are you?
	// S: Fine
	// C: Bye
	// S: Cya
}
