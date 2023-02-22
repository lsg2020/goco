package main

import (
	"context"
	"log"
	"time"

	co "github.com/lsg2020/goco"
)

func main() {
	coroutine, err := co.New(nil)
	if err != nil {
		log.Fatalln(err)
	}
	testData := 0

	for i := 0; i < 20; i++ {
		go func(ctx context.Context, coID int) {
			var result int

			ctx, _ = context.WithTimeout(ctx, time.Second*3)
			err := coroutine.Run(ctx, func(ctx context.Context) error {
				for i := 0; i < 5; i++ {
					testData++
					log.Printf("current co:%d i:%d value:%d", coID, i, testData)

					co.Sleep(ctx, time.Second)
				}

				result = testData
				return nil
			}, nil)

			log.Println("run result:", coID, result, err)
		}(context.Background(), i)
	}

	time.Sleep(time.Second * 10)
}
