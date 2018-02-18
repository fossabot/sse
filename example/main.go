package main

import (
	"net/http"
	"time"

	"github.com/davidsbond/sse"
)

func main() {
	cnf := sse.Config{
		Timeout:   time.Second * 3,
		Tolerance: 3,
	}

	broker := sse.NewBroker(cnf)

	http.HandleFunc("/connect", broker.ClientHandler)
	http.HandleFunc("/broadcast", broker.EventHandler)

	data := []byte("hello world")

	broker.Broadcast(data)

	http.ListenAndServe(":8080", nil)
}
