package compute // import "github.com/docker/docker/pkg/compute"

import "context"

// TODO(rvolosatovs): Refactor into Result[T] once support for generics lands.

// Result is a result of a computation.
type Result struct {
	Value interface{}
	Error error
}

func NewSingleton() *Singleton {
	return &Singleton{
		callCh:   make(chan struct{}, 1),
		resultCh: make(chan Result),
	}
}

type Singleton struct {
	callCh   chan struct{}
	resultCh chan Result
}

func (s Singleton) Do(ctx context.Context, f func(context.Context) (interface{}, error)) (interface{}, error) {
	select {
	case s.callCh <- struct{}{}:
		// Lock acquired - perform computation.

	case res := <-s.resultCh:
		// Another goroutine computed the result - return.
		return res.Value, res.Error

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var res Result
	v, err := f(ctx)
	if err != nil {
		res = Result{
			Error: err,
		}
	} else {
		res = Result{
			Value: v,
		}
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case s.resultCh <- res:
			// Push computation result to other goroutines calling the function, if any.

		default:
			// Release lock.
			<-s.callCh
			return res.Value, res.Error
		}
	}
}
