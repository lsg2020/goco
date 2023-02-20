package main

import (
	"context"
	"fmt"
	"log"
	"time"

	co "github.com/lsg2020/goco"
)

func sleep(ctx context.Context) error {
	time.Sleep(time.Second)
	return nil
}

func main() {
	coroutine, err := co.New(nil)
	if err != nil {
		log.Fatalln(err)
	}
	testData := 0

	for i := 0; i < 20; i++ {
		go func(ctx context.Context, coID int) {
			var result int

			err := coroutine.Run(ctx, func(ctx context.Context) error {
				for i := 0; i <= 5; i++ {
					testData++
					log.Printf("current co:%d i:%d value:%d", coID, i, testData)

					err = co.Await(ctx, sleep)
					if err != nil {
						return fmt.Errorf("async run failed, %w", err)
					}
				}

				result = testData
				return nil
			}, nil)

			log.Println("run result:", coID, result, err)
		}(context.Background(), i)
	}

	time.Sleep(time.Second * 10)
}
