package broker_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davidsbond/sse/broker"
	"github.com/stretchr/testify/assert"
)

type (
	TestRecorder struct {
		close   chan bool
		header  http.Header
		data    []byte
		code    int
		flushed []byte
	}
)

func (tr TestRecorder) CloseNotify() <-chan bool {
	return tr.close
}

func (tr TestRecorder) Header() http.Header {
	return tr.header
}

func (tr TestRecorder) Write(data []byte) (int, error) {
	tr.data = data

	return len(data), nil
}

func (tr TestRecorder) WriteHeader(code int) {
	tr.code = code
}

func (tr TestRecorder) Flush() {
	tr.flushed = tr.data
}

func TestBroker_New(t *testing.T) {
	tt := []struct {
		Timeout   time.Duration
		Tolerance int
	}{
		{Timeout: time.Second, Tolerance: 3},
	}

	for _, tc := range tt {
		broker := broker.New(tc.Timeout, tc.Tolerance, nil)

		assert.NotNil(t, broker)
	}
}

func TestBroker_Handlers(t *testing.T) {
	tt := []struct {
		Timeout            time.Duration
		Tolerance          int
		ContentType        string
		ExpectedError      string
		Recorder           http.ResponseWriter
		Data               []byte
		ExpectedCode       int
		AssertErrorHandler bool
	}{
		{
			Timeout:      time.Second,
			Tolerance:    3,
			ContentType:  "text/event-stream",
			Recorder:     &TestRecorder{header: http.Header{}},
			ExpectedCode: http.StatusOK,
		},
		{
			Timeout:      time.Second,
			Tolerance:    3,
			ContentType:  "text/event-stream",
			Recorder:     &TestRecorder{header: http.Header{}},
			ExpectedCode: http.StatusOK,
		},
		{
			Timeout:            time.Second,
			Tolerance:          3,
			ContentType:        "text/event-stream",
			Recorder:           httptest.NewRecorder(),
			ExpectedError:      "client does not support streaming",
			AssertErrorHandler: true,
		},
		{
			Timeout:       time.Second,
			Tolerance:     3,
			ContentType:   "text/event-stream",
			Recorder:      httptest.NewRecorder(),
			ExpectedError: "client does not support streaming",
		},
	}

	for _, tc := range tt {
		var handler broker.ErrorHandler

		if tc.AssertErrorHandler {
			handler = func(w http.ResponseWriter, r *http.Request, err error) {
				assert.Contains(t, err.Error(), tc.ExpectedError)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

		// Create a new broker
		broker := broker.New(tc.Timeout, tc.Tolerance, handler)

		// The test recorder allows us to cast to http.Flusher & http.CloseNotifier
		w := tc.Recorder

		// Create the request
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Add("Content-Type", tc.ContentType)

		// Connect to the broker, give it 1 second to create the
		// client
		go broker.ClientHandler(w, r)
		<-time.Tick(time.Second)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", bytes.NewBuffer(tc.Data))

		// Post an event.
		broker.EventHandler(rec, req)

		if tc.ExpectedCode > 0 {
			assert.Equal(t, tc.ExpectedCode, rec.Code)
		}
	}
}

func TestBroker_Broadcast(t *testing.T) {
	tt := []struct {
		Timeout       time.Duration
		Tolerance     int
		ContentType   string
		ExpectedError string
		Data          []byte
	}{
		{
			Timeout:     time.Second,
			Tolerance:   3,
			ContentType: "text/event-stream",
		},
	}

	for _, tc := range tt {
		// Create a new broker
		broker := broker.New(tc.Timeout, tc.Tolerance, nil)

		// The test recorder allows us to cast to http.Flusher & http.CloseNotifier
		w := &TestRecorder{header: http.Header{}}

		// Create the request
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Add("Content-Type", tc.ContentType)

		// Connect to the broker, give it 1 second to create the
		// client
		go broker.ClientHandler(w, r)
		<-time.Tick(time.Second)

		if err := broker.Broadcast(tc.Data); err != nil {
			assert.Contains(t, err.Error(), tc.ExpectedError)
		}
	}
}

func TestBroker_BroadcastTo(t *testing.T) {
	tt := []struct {
		Timeout       time.Duration
		Tolerance     int
		ContentType   string
		ExpectedError string
		ClientID      string
		IDParam       string
		Data          []byte
	}{
		{
			Timeout:     time.Second,
			Tolerance:   3,
			ContentType: "text/event-stream",
			ClientID:    "1234",
			IDParam:     "1234",
		},
		{
			Timeout:       time.Second,
			Tolerance:     3,
			ContentType:   "text/event-stream",
			ClientID:      "",
			IDParam:       "1234",
			ExpectedError: "no client with id",
		},
	}

	for _, tc := range tt {
		// Create a new broker
		broker := broker.New(tc.Timeout, tc.Tolerance, nil)

		// The test recorder allows us to cast to http.Flusher & http.CloseNotifier
		w := &TestRecorder{header: http.Header{}}

		// Create the request
		r := httptest.NewRequest("GET", "/connect?id="+tc.IDParam, nil)
		r.Header.Add("Content-Type", tc.ContentType)

		// Connect to the broker, give it 1 second to create the
		// client
		go broker.ClientHandler(w, r)
		<-time.Tick(time.Second)

		if err := broker.BroadcastTo(tc.ClientID, tc.Data); err != nil {
			assert.Contains(t, err.Error(), tc.ExpectedError)
		}
	}
}
