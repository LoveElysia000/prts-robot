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

func TestConcurrency(t *testing.T) {
	pool := NewWorkerPool(2)
	defer pool.Shutdown()

	results := make(chan string, 3)

	submitTask := func(label string, delay time.Duration) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			reply, err := pool.Submit(ctx, &Task{
				Priority: PriorityLight,
				Handler: func(ctx context.Context) (string, error) {
					time.Sleep(delay)
					return label, nil
				},
			})
			if err != nil {
				results <- "err: " + err.Error()
				return
			}
			results <- reply
		}()
	}

	submitTask("task-1", 100*time.Millisecond)
	submitTask("task-2", 100*time.Millisecond)
	submitTask("task-3", 10*time.Millisecond) // queued, waits for a worker

	var completed []string
	for i := 0; i < 3; i++ {
		select {
		case r := <-results:
			completed = append(completed, r)
		case <-time.After(2 * time.Second):
			t.Fatal("test timed out")
		}
	}
	if len(completed) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(completed), completed)
	}
}

func TestShutdown(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Shutdown()

	ctx := context.Background()
	_, err := pool.Submit(ctx, &Task{
		Priority: PriorityLight,
		Handler:  func(ctx context.Context) (string, error) { return "ok", nil },
	})
	if err == nil {
		t.Error("expected error after shutdown")
	}
}

func TestWorkerPanic(t *testing.T) {
	pool := NewWorkerPool(1)
	defer pool.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := pool.Submit(ctx, &Task{
		Priority: PriorityLight,
		Handler: func(ctx context.Context) (string, error) {
			panic("unexpected")
		},
	})
	if err == nil {
		t.Error("expected error from panicking handler")
	}
}
