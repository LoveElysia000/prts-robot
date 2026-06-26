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

func TestPriority(t *testing.T) {
	pool := NewWorkerPool(2)
	defer pool.Shutdown()

	results := make(chan string, 3)

	submitTask := func(prio TaskPriority, label string, delay time.Duration) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			reply, err := pool.Submit(ctx, &Task{
				Priority: prio,
				Handler: func(ctx context.Context) (string, error) {
					time.Sleep(delay)
					return label, nil
				},
				OnStart: func() {},
			})
			if err != nil {
				results <- "err: " + err.Error()
				return
			}
			results <- reply
		}()
	}

	// Fill both workers with heavy tasks
	submitTask(PriorityHeavy, "heavy-1", 50*time.Millisecond)
	submitTask(PriorityHeavy, "heavy-2", 300*time.Millisecond)
	time.Sleep(10 * time.Millisecond) // ensure they're dequeued

	// Submit a light task — must wait for a worker
	// When first heavy finishes (~200ms), worker prefers lightCh
	// so light-1 should finish before heavy-2 (which has ~200ms remaining on worker 2)
	submitTask(PriorityLight, "light-1", 10*time.Millisecond)

	var order []string
	for i := 0; i < 3; i++ {
		select {
		case r := <-results:
			order = append(order, r)
		case <-time.After(2 * time.Second):
			t.Fatal("test timed out")
		}
	}

	// Expected: heavy-1 finishes first (~200ms), then light-1 (~10ms more),
	// then heavy-2 (~200ms on its own worker). So: heavy-1, light-1, heavy-2.
	// Key assertion: light-1 must finish before heavy-2 (priority worked)
	lightIdx, heavy2Idx := -1, -1
	for i, v := range order {
		if v == "light-1" {
			lightIdx = i
		}
		if v == "heavy-2" {
			heavy2Idx = i
		}
	}
	if lightIdx < 0 || heavy2Idx < 0 {
		t.Fatalf("missing results, got order: %v", order)
	}
	if lightIdx > heavy2Idx {
		t.Errorf("light-1 should finish before heavy-2 (priority), got order: %v", order)
	}
}

func TestShutdown(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Shutdown()

	ctx := context.Background()
	_, err := pool.Submit(ctx, &Task{
		Priority: PriorityLight,
		Handler:  func(ctx context.Context) (string, error) { return "ok", nil },
		OnStart:  func() {},
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
		OnStart: func() {},
	})
	if err == nil {
		t.Error("expected error from panicking handler")
	}
}