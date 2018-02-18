package client_test

import (
	"testing"
	"time"

	"github.com/davidsbond/sse/client"
)

func BenchmarkClient_Write(b *testing.B) {
	b.StopTimer()
	client := client.New(time.Second, 3, "test")

	go func() {
		for {
			<-client.Listen()
		}
	}()

	data := make([]byte, 1024)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		client.Write(data)
	}
}

func BenchmarkClient_Listen(b *testing.B) {
	b.StopTimer()
	client := client.New(time.Second, 3, "test")
	data := make([]byte, 1024)

	go func() {
		for i := 0; i < b.N; i++ {
			client.Write(data)
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		<-client.Listen()
	}
}
