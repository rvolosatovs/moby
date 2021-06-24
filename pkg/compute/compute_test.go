package compute_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	. "github.com/docker/docker/pkg/compute"
)

func TestSingleton(t *testing.T) {
	s := NewSingleton()

	assert := func(ctx context.Context, f func(ctx context.Context) (interface{}, error), expectedValue interface{}, expectedError error) {
		v, err := s.Do(ctx, f)
		if err != expectedError || v != expectedValue {
			t.Errorf(`actual: (value: %v; error: %v)
expected: (value: %v; error: %v)`,
				v, err,
				expectedValue, expectedError)
		}
	}

	const testValue = 42
	testError := errors.New("test error")

	makeIDFunc := func(v interface{}, err error) func(context.Context) (interface{}, error) {
		return func(context.Context) (interface{}, error) { return v, err }
	}
	testErrorIDFunc := makeIDFunc(nil, testError)
	testValueIDFunc := makeIDFunc(testValue, nil)

	assert(context.Background(), testErrorIDFunc, nil, testError)
	assert(context.Background(), testValueIDFunc, testValue, nil)

	startCh := make(chan struct{})
	doneCh := make(chan struct{})

	baseWG := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		baseWG.Add(1)
		go func() {
			assert(context.Background(), func(context.Context) (interface{}, error) {
				close(startCh)

				<-doneCh
				return testValue, nil
			}, testValue, nil)
			baseWG.Done()
		}()
	}

	select {
	case <-startCh:

	case <-time.After(time.Second):
		t.Fatalf("test timeout")
	}

	noCallFunc := func(context.Context) (interface{}, error) { t.Error("should not be called"); return nil, nil }

	cancelWG := &sync.WaitGroup{}
	cancelCtx, cancel := context.WithCancel(context.Background())
	for i := 0; i < 10; i++ {
		cancelWG.Add(1)
		go func() {
			assert(cancelCtx, noCallFunc, nil, context.Canceled)
			cancelWG.Done()
		}()
	}
	time.Sleep(time.Nanosecond)
	cancel()
	cancelWG.Wait()

	select {
	case doneCh <- struct{}{}:

	case <-time.After(time.Second):
		t.Fatalf("test timeout")
	}
	baseWG.Wait()
}
