package core

import (
	"context"
	"testing"
	"time"
)

func TestSubmitLight(t *testing.T) {
	pool := NewWorkerPool(1)
	defer pool.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan string, 1)
	go func() {
		reply, err := pool.Submit(ctx, &Task{
			Priority: PriorityLight,
			Handler: func(ctx context.Context) (string, error) {
				return "hello", nil
			},
			OnStart: func() {},
		})
		if err != nil {
			done <- "err: " + err.Error()
			return
		}
		done <- reply
	}()

	select {
	case result := <-done:
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Submit timed out")
	}
}
