package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

type Server struct {
	conns map[*websocket.Conn]bool
}

type indexHandler struct {
}

func (ih indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.ParseFiles("index.html"))

	err := tpl.Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}

}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
	}
}

// handle incoming client connection
func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("new incoming connection from client:", ws.RemoteAddr())

	// use mutex here as maps are not concurrent safe
	s.conns[ws] = true

	// read from each connection
	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			// if connection has closed its own connection then close the loop
			if err == io.EOF {
				break
			}
			fmt.Println("read error:", err)
			continue
		}
		msg := buf[:n]

		s.broadcast(msg)
	}
}

// broadcast the message to each client
func (s *Server) broadcast(b []byte) {
	for ws := range s.conns {
		// we write to each connection
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(b); err != nil {
				fmt.Println("Broadcast error: ", err)
			}
		}(ws)
	}
}

func main() {
	server := NewServer()
	ih := indexHandler{}
	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.Handle("/", ih)

	// listenandserve will block, so send in a goroutine
	go func() {
		err := http.ListenAndServe(":3000", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Println("Listening on port :3000")

	// block main from closing
	ch := make(chan int)
	<-ch
}
