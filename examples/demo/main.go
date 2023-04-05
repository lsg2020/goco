package main

import (
	"context"
	"log"
	"time"

	co "github.com/lsg2020/goco"
)

func main() {
	ex, err := co.NewExecuter(context.Background(), &co.ExOptions{Name: "test"})
	if err != nil {
		log.Fatalln(err)
	}
	coroutine, err := co.New(&co.Options{Name: "test", DebugInfo: "demo", Executer: ex})
	if err != nil {
		log.Fatalln(err)
	}
	testData := 0

	for i := 0; i < 20; i++ {
		go func(ctx context.Context, coID int) {
			var result int

			ctx, _ = context.WithTimeout(ctx, time.Second*3)
			err := coroutine.RunSync(ctx, func(ctx context.Context) error {
				for i := 0; i < 5; i++ {
					testData++
					log.Printf("current co:%d i:%d value:%d", coID, i, testData)

					co.Sleep(ctx, time.Second)
				}

				result = testData
				return nil
			}, &co.RunOptions{})

			log.Println("run result:", coID, result, err)
		}(context.Background(), i)
	}

	time.Sleep(time.Second * 10)
}
