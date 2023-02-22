package co

type Options struct {
	Parent          *Coroutine
	AsyncTaskSubmit func(func()) error
}

type RunOptions struct {
}
