# goco
async/await, thread safe coroutine for golang

# usage
* create corotuine 
```go
coroutine, err := co.New(&co.Options{Name: "test"})
```
* run coroutine task in goroutine
```go
amount := 0
err := coroutine.RunAsync(ctx, func(ctx context.Context) error {
    for i := 0; i<10; i++ {
        amount++ // thread safe
        co.Sleep(ctx, time.Second)
    }
    return nil
}, nil)

result := coroutine.RunSync(ctx, func(ctx context.Context) error {
    for i := 0; i<10; i++ {
        amount++ // thread safe
        co.Sleep(ctx, time.Second)
    }
    return nil
}, nil)
```
