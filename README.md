# sse

[![CircleCI](https://img.shields.io/circleci/project/github/davidsbond/sse.svg)](https://circleci.com/gh/davidsbond/sse)
[![GoDoc](https://godoc.org/github.com/davidsbond/sse?status.svg)](http://godoc.org/github.com/davidsbond/sse)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidsbond/sse)](https://goreportcard.com/report/github.com/davidsbond/sse)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/davidsbond/sse/release/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fdavidsbond%2Fsse.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fdavidsbond%2Fsse?ref=badge_shield)

A golang library for implementing a [Server Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) broker

## usage

```go
    // Create a configuration for the SSE broker
    config := sse.Config{
        Timeout: time.Duration * 3,
        Tolerance: 3,
        ErrorHandler: nil,
    }

    // Create a broker
    broker := sse.NewBroker(config)

    // Register the broker's HTTP handlers to your
    // router of choice.
    http.HandleFunc("/connect", broker.ClientHandler)
    http.HandleFunc("/broadcast", broker.EventHandler)

    // Programatically create events
    broker.Broadcast([]byte("hello world"))
    broker.BroadcastTo("123", []byte("hello world"))
```

## listening for events

```javascript
    // Connect to the event broker
    const source = new EventSource("http://localhost:8080/connect");

    // Optionally, supply a custom identifier for messaging individual clients
    // const source = new EventSource("http://localhost:8080/connect?id=1234");

    // Listen for incoming events
    source.onmessage = (event) => {
        // Do something with the event data
    };
```

## custom error handlers

If you want any HTTP errors returned to be in a certain format, you can supply a custom error handler to the broker

```go
    handler := func(w http.ResponseWriter, r *http.Request, err error) {
        // Write whatever you like to 'w'
    }

    // Create a configuration for the SSE broker
    config := sse.Config{
        Timeout: time.Duration * 3,
        Tolerance: 3,
        ErrorHandler: handler,
    }

    // Create a broker
    broker := sse.NewBroker(config)
```