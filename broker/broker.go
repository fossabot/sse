// Package broker contains types to be used to host a Server Sent Events broker.
package broker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/davidsbond/sse/client"
)

type (
	// The Broker interface describes the Server Side Events broker, propagating messages
	// to all connected clients.
	Broker interface {
		Broadcast(data []byte) error
		BroadcastTo(id string, data []byte) error
		ClientHandler(w http.ResponseWriter, r *http.Request)
		EventHandler(w http.ResponseWriter, r *http.Request)
	}

	// ErrorHandler is a convenience wrapper for the HTTP error handling function.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	defaultBroker struct {
		timeout      time.Duration
		clients      *sync.Map
		errorHandler ErrorHandler
		tolerance    int
	}
)

// New creates a new instance of the Broker type. The 'timeout' parameter determines how long
// the broker will wait to write a message to a client, if this timeout is exceeded, the client
// will not recieve that message. The 'tolerance' parameter indicates how many sequential errors
// can occur when communicating with a client until the client is forcefully disconnected. The
// 'eh' parameter is a custom HTTP error handler that the broker will use when HTTP errors are
// raised. If 'eh' is null, the default http.Error method is used.
func New(timeout time.Duration, tolerance int, eh ErrorHandler) Broker {
	return &defaultBroker{
		timeout:      timeout,
		clients:      &sync.Map{},
		tolerance:    tolerance,
		errorHandler: eh,
	}
}

func (b *defaultBroker) BroadcastTo(id string, data []byte) error {
	item, ok := b.clients.Load(id)

	if !ok {
		return fmt.Errorf("no client with id %v exists", id)
	}

	client, ok := item.(*client.Client)

	if !ok {
		b.removeClient(id)
		return errors.New("client is malformed, disconnecting")
	}

	return client.Write(data)
}

// Broadcast writes the given data to all connected clients. If a client exceeds its error tolerance, it is
// forcefully disconnected from the broker. All errors are concatenated with newlines and returned from this
// method as a single error.
func (b *defaultBroker) Broadcast(data []byte) error {
	var out []string

	// Loop through each connected client.
	b.clients.Range(func(key, value interface{}) bool {
		client, ok := value.(*client.Client)

		// If we couldn't cast the client, something strange has
		// gotten into the map. Add an error to the array and
		// force disconnect the client.
		if !ok {
			err := fmt.Errorf("found malformed client with id %v, disconnecting", key)
			out = append(out, err.Error())
			b.clients.Delete(key)
		}

		// Attempt to write data to the client
		if err := client.Write(data); err != nil {
			// If an error occured, check if we should force
			// disconnect the client.
			if client.ShouldDisconnect() {
				b.removeClient(client.ID())
			}

			out = append(out, err.Error())
		}

		return true
	})

	// If we have multiple errors, concatenate them with newlines.
	if len(out) > 0 {
		return errors.New(strings.Join(out, "\n"))
	}

	return nil
}

// EventHandler is an HTTP handler that allows a client to broadcast an event to the
// broker. This method should be registered to an endpoint of your choosing. For information
// on error handling, see the broker.SetErrorHandler method.
//
// Example using http (https://golang.org/pkg/net/http/)
//
// http.HandleFunc("/broadcast", broker.EventHandler)
// http.ListenAndServe(":8080")
//
// Example using Mux (https://github.com/gorilla/mux)
//
// r := mux.NewRouter()
// r.HandleFunc("/broadcast", broker.EventHandler).Methods("POST")
//
// http.ListenAndServe(":8080", r)
func (b *defaultBroker) EventHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to read the provided event data.
	data, err := ioutil.ReadAll(r.Body)

	// If we fail to read, either use the custom error handler or
	// use the default http error.
	if err != nil {
		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	id := r.URL.Query().Get("id")

	// Attempt to broadcast the event data to the connected clients. If this
	// fails, use either the custom error handler or the default http handler.
	if id != "" {
		err = b.BroadcastTo(id, data)
	} else {
		err = b.Broadcast(data)
	}

	if err != nil {
		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ClientHandler is an HTTP handler that allows a client to connect to the
// broker. This method should be registered to an endpoint of your choosing.
// For information on error handling, see the broker.SetErrorHandler method.
//
// Example using http (https://golang.org/pkg/net/http/)
//
// http.HandleFunc("/connect", broker.ClientHandler)
// http.ListenAndServe(":8080")
//
// Example using Mux (https://github.com/gorilla/mux)
//
// r := mux.NewRouter()
// r.HandleFunc("/connect", broker.ClientHandler).Methods("GET")
//
// http.ListenAndServe(":8080", r)
func (b *defaultBroker) ClientHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to cast the response writer to a flusher & close notifier
	flusher, ok := w.(http.Flusher)
	notify, ok := w.(http.CloseNotifier)

	if !ok {
		// If we fail to cast, use the custom error handler if set. Otherwise,
		// use the default http error handler.
		err := errors.New("client does not support streaming")

		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	// Set the required headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a new client with the configured timeout &
	// tolerance.
	client := client.New(b.timeout, b.tolerance, r.URL.Query().Get("id"))
	id := client.ID()

	// Ensure that no custom identifiers collide.
	if b.hasClient(id) {
		err := fmt.Errorf("a client with id %v already exists", id)

		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	defer b.removeClient(id)
	b.addClient(client)

	// Listen if the client disconnects.
	close := notify.CloseNotify()
	go b.listenForClose(id, close)

	// While the client is connected
	for b.hasClient(id) {
		select {
		// If we read an event, write it to the client
		case data := <-client.Listen():
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			break

		// If we exceed the timeout, continue.
		case <-time.Tick(b.timeout):
			continue
		}
	}
}

func (b *defaultBroker) addClient(client *client.Client) {
	b.clients.Store(client.ID(), client)
}

func (b *defaultBroker) removeClient(id string) {
	b.clients.Delete(id)
}

func (b *defaultBroker) listenForClose(id string, notify <-chan bool) {
	<-notify
	b.removeClient(id)
}

func (b *defaultBroker) hasClient(id string) bool {
	_, ok := b.clients.Load(id)

	return ok
}

func (b *defaultBroker) httpError(w http.ResponseWriter, r *http.Request, err error, code int) {
	if b.errorHandler != nil {
		b.errorHandler(w, r, err)
		return
	}

	http.Error(w, err.Error(), code)
}
