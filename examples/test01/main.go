package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"time"

	co "github.com/lsg2020/goco"
)

func main() {
	go http.ListenAndServe(":9090", nil)

	ex, err := co.NewExecuter(context.Background(), &co.ExOptions{Name: "test"})
	if err != nil {
		log.Fatalln(err)
	}
	coroutine, err := co.New(&co.Options{Name: "test", DebugInfo: "test01", Executer: ex})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*600)
	defer cancel()
	ch := make(chan struct{}, 100000)

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
			}, &co.RunOptions{Result: func(err error) {
				<-ch
			}, RunLimitTime: -1})
		case <-ctx.Done():
			return
		}
	}
}
