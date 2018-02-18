package sse_test

import (
	"net/http"
	"time"

	"github.com/davidsbond/sse"
)

func ExampleSSE_NewBroker() {
	// Create a configuration for the broker
	cnf := sse.Config{
		Timeout:   time.Second * 5,
		Tolerance: 3,

		// Optionally, you can provide a custom HTTP error handler to
		// return errors in whichever way you please.
		ErrorHandler: nil,
	}

	// Create a new broker
	broker := sse.NewBroker(cnf)

	// Register the client & event HTTP handlers
	http.HandleFunc("/connect", broker.ClientHandler)
	http.HandleFunc("/broadcast", broker.EventHandler)

	// Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
