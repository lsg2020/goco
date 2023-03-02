package co

import (
	"context"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestCO(t *testing.T) {
	coroutine, err := New(&Options{Name: "test", DebugInfo: "test"})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ch := make(chan struct{}, 10000)

	amount := 0
	for {
		select {
		case ch <- struct{}{}:
			coroutine.RunAsync(ctx, func(ctx context.Context) error {
				t := time.Duration(rand.Int()%3+1) * time.Second
				begin := time.Now()

				for time.Now().Sub(begin) < t {
					amount++
					if amount%100000 == 0 {
						log.Println("==============", amount)
					}
					coroutine.Sleep(ctx, time.Millisecond*100)
				}
				return nil
			}, &RunOptions{Result: func(err error) {
				<-ch
			}})
		case <-ctx.Done():
			return
		}
	}
}
