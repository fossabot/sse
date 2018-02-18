package sse_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/davidsbond/sse"
	"github.com/stretchr/testify/assert"
)

func TestSSE_NewBroker(t *testing.T) {
	tt := []struct {
		UseHandler bool
		Timeout    time.Duration
		Tolerance  int
	}{
		{UseHandler: true, Timeout: time.Second, Tolerance: 3},
		{Timeout: time.Second, Tolerance: 3},
	}

	for _, tc := range tt {
		cnf := sse.Config{
			Timeout:   tc.Timeout,
			Tolerance: tc.Tolerance,
		}

		if tc.UseHandler {
			cnf.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {}
		}

		broker := sse.NewBroker(cnf)

		assert.NotNil(t, broker)
	}
}
