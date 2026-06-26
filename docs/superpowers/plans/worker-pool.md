# Worker Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `goroutine-per-message + llmSem` with a bounded WorkerPool providing priority scheduling and progress feedback.

**Architecture:** Dual-channel priority queue (lightCh + heavyCh) with nested `select` in 3 workers. `Submit(ctx, task)` blocks until completion — caller wraps in goroutine + manages Discord placeholder/edit.

**Tech Stack:** Go 1.25, sync, context

---

### Task 1: Core types + light-task Submit test

**Files:**
- Create: `internal/core/worker.go`
- Create: `internal/core/worker_test.go`

- [ ] **Step 1: Write failing test for Submit (light task)**

```go
// internal/core/worker_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/Ein/project2/robot && go test ./internal/core/ -run TestSubmitLight -v`
Expected: FAIL — `undefined: NewWorkerPool`

- [ ] **Step 3: Write minimal types + constructor in worker.go**

```go
// internal/core/worker.go
package core

import (
    "context"
    "fmt"
    "log/slog"
    "sync"
)

type TaskPriority int

const (
    PriorityLight TaskPriority = 1
    PriorityHeavy TaskPriority = 2
)

type Task struct {
    Priority TaskPriority
    Handler  func(ctx context.Context) (string, error)
    OnStart  func()
}

type WorkerPool struct {
    lightCh chan *Task
    heavyCh chan *Task
    wg      sync.WaitGroup
    ctx     context.Context
    cancel  context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    p := &WorkerPool{
        lightCh: make(chan *Task, 64),
        heavyCh: make(chan *Task, 64),
        ctx:     ctx,
        cancel:  cancel,
    }
    for i := 0; i < workers; i++ {
        p.wg.Add(1)
        go p.run(i)
    }
    return p
}

func (p *WorkerPool) Submit(ctx context.Context, task *Task) (string, error) {
    ch := p.lightCh
    if task.Priority == PriorityHeavy {
        ch = p.heavyCh
    }
    select {
    case ch <- task:
    case <-ctx.Done():
        return "", ctx.Err()
    case <-p.ctx.Done():
        return "", fmt.Errorf("pool is shutting down")
    }

    // Wait for result via a channel. Worker sends result back.
    // For now, we need a result channel in Task.
    // ... (will complete in next step)
    return "", fmt.Errorf("not implemented")
}

func (p *WorkerPool) Shutdown() {
    p.cancel()
    p.wg.Wait()
}

func (p *WorkerPool) run(id int) {
    defer p.wg.Done()
    slog.Info("worker started", "id", id)
    // ... (will complete in next step)
}
```

- [ ] **Step 4: Build to verify types compile**

Run: `cd /Users/Ein/project2/robot && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/core/worker.go internal/core/worker_test.go
git commit -m "wip: WorkerPool types + failing Submit test"
```

---

### Task 2: Result channel + working Submit

**Files:**
- Modify: `internal/core/worker.go`
- Modify: `internal/core/worker_test.go`

- [ ] **Step 1: Add result channel to Task, implement worker loop**

Replace `worker.go` with:

```go
package core

import (
    "context"
    "fmt"
    "log/slog"
    "sync"
)

type TaskPriority int

const (
    PriorityLight TaskPriority = 1
    PriorityHeavy TaskPriority = 2
)

type Task struct {
    Priority TaskPriority
    Handler  func(ctx context.Context) (string, error)
    OnStart  func()
    resultCh chan taskResult
}

type taskResult struct {
    reply string
    err   error
}

type WorkerPool struct {
    lightCh chan *Task
    heavyCh chan *Task
    wg      sync.WaitGroup
    ctx     context.Context
    cancel  context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    p := &WorkerPool{
        lightCh: make(chan *Task, 64),
        heavyCh: make(chan *Task, 64),
        ctx:     ctx,
        cancel:  cancel,
    }
    for i := 0; i < workers; i++ {
        p.wg.Add(1)
        go p.run(i)
    }
    return p
}

func (p *WorkerPool) Submit(ctx context.Context, task *Task) (string, error) {
    task.resultCh = make(chan taskResult, 1)
    ch := p.lightCh
    if task.Priority == PriorityHeavy {
        ch = p.heavyCh
    }
    select {
    case ch <- task:
    case <-ctx.Done():
        return "", ctx.Err()
    case <-p.ctx.Done():
        return "", fmt.Errorf("pool is shutting down")
    }

    select {
    case result := <-task.resultCh:
        return result.reply, result.err
    case <-ctx.Done():
        return "", ctx.Err()
    }
}

func (p *WorkerPool) Shutdown() {
    p.cancel()
    p.wg.Wait()
}

func (p *WorkerPool) run(id int) {
    defer p.wg.Done()
    defer func() {
        if r := recover(); r != nil {
            slog.Error("worker panic", "id", id, "panic", r)
        }
    }()
    slog.Info("worker started", "id", id)
    for {
        var task *Task
        // Nested select: light priority
        select {
        case task = <-p.lightCh:
        default:
            select {
            case task = <-p.lightCh:
            case task = <-p.heavyCh:
            case <-p.ctx.Done():
                return
            }
        }
        if task == nil {
            continue
        }
        if task.OnStart != nil {
            task.OnStart()
        }
        reply, err := task.Handler(p.ctx)
        task.resultCh <- taskResult{reply: reply, err: err}
    }
}
```

- [ ] **Step 2: Run the test**

Run: `cd /Users/Ein/project2/robot && go test ./internal/core/ -run TestSubmitLight -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/worker.go
git commit -m "feat: WorkerPool Submit with result channel"
```

---

### Task 3: Priority queue tests

**Files:**
- Modify: `internal/core/worker_test.go`

- [ ] **Step 1: Write priority test**

```go
func TestPriority(t *testing.T) {
    pool := NewWorkerPool(2)
    defer pool.Shutdown()

    results := make(chan string, 4)

    // Submit 2 heavy tasks (slow) then 2 light tasks (fast)
    submitTask := func(prio TaskPriority, label string, delay time.Duration) {
        go func() {
            ctx := context.Background()
            reply, _ := pool.Submit(ctx, &Task{
                Priority: prio,
                Handler: func(ctx context.Context) (string, error) {
                    time.Sleep(delay)
                    return label, nil
                },
                OnStart: func() {},
            })
            results <- reply
        }()
    }

    submitTask(PriorityHeavy, "heavy-1", 200*time.Millisecond)
    submitTask(PriorityHeavy, "heavy-2", 200*time.Millisecond)
    time.Sleep(10 * time.Millisecond) // ensure heavies are dequeued first
    submitTask(PriorityLight, "light-1", 10*time.Millisecond)
    submitTask(PriorityLight, "light-2", 10*time.Millisecond)

    // Lights should finish before heavies because nested select prefers lightCh
    // With 2 workers and 2 heavy tasks blocking, lights must wait for a worker to free
    // BUT once a worker finishes a heavy, it checks lightCh first before next heavy
    var order []string
    for i := 0; i < 4; i++ {
        select {
        case r := <-results:
            order = append(order, r)
        case <-time.After(2 * time.Second):
            t.Fatal("test timed out")
        }
    }

    // Verify: light-1 and light-2 should appear before heavy tasks finish
    lightPos1, lightPos2 := -1, -1
    for i, v := range order {
        if v == "light-1" {
            lightPos1 = i
        }
        if v == "light-2" {
            lightPos2 = i
        }
    }
    if lightPos1 >= 2 || lightPos2 >= 2 {
        t.Errorf("light tasks should complete before heavy tasks, got order: %v", order)
    }
}
```

- [ ] **Step 2: Run test**

Run: `cd /Users/Ein/project2/robot && go test ./internal/core/ -run TestPriority -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/worker_test.go
git commit -m "test: WorkerPool priority queue behavior"
```

---

### Task 4: Shutdown + error handling tests

**Files:**
- Modify: `internal/core/worker_test.go`

- [ ] **Step 1: Write shutdown and panic tests**

```go
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
```

- [ ] **Step 2: Run tests**

Run: `cd /Users/Ein/project2/robot && go test ./internal/core/ -run "TestShutdown|TestWorkerPanic" -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/worker_test.go
git commit -m "test: WorkerPool shutdown and panic recovery"
```

---

### Task 5: Integrate into bot.go

**Files:**
- Modify: `internal/core/bot.go`

- [ ] **Step 1: Add WorkerPool field to Bot, replace dispatch logic**

In `bot.go`:

Add to Bot struct:
```go
type Bot struct {
    cfg     *Config
    llm     *llm.Client
    session *session.Manager
    dg      *discordgo.Session
    persona *persona.Manager
    pool    *WorkerPool  // NEW
}
```

In `NewBot`, after `personaMgr`:
```go
pool := NewWorkerPool(cfg.Worker.Count)
```

Add to return:
```go
return &Bot{
    ...
    pool: pool,
}
```

Replace the goroutine dispatch in `handleMessage`:

At the pool-bound message dispatch (after `s.ChannelTyping` is set and message is classified):

```go
// For pool-bound messages (chat and slow commands):
go func() {
    msg, _ := s.ChannelMessageSendReply(m.ChannelID, "⏳ 排队中...", m.Reference())
    submitCtx, submitCancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer submitCancel()

    task := &Task{
        Priority: classifyTask(cmdText), // Light for chat/correct, Heavy for generate
        Handler: func(ctx context.Context) (string, error) {
            return b.callLLM(ctx, sessionKey, messages)
        },
        OnStart: func() {
            s.ChannelTyping(m.ChannelID)
        },
    }

    reply, err := b.pool.Submit(submitCtx, task)
    if err != nil {
        s.ChannelMessageEdit(m.ChannelID, msg.ID, "抱歉，处理超时，请稍后再试。")
        return
    }
    s.ChannelMessageEdit(m.ChannelID, msg.ID, reply)
}()

// classifyTask returns PriorityHeavy for /生成角色, PriorityLight for everything else.
func classifyTask(cmdText string) TaskPriority {
    if strings.HasPrefix(cmdText, "/生成角色") {
        return PriorityHeavy
    }
    return PriorityLight
}
```

Remove the old `llmSem` variable declaration and all `llmSem <-` / `<-llmSem` usage.

Also strip the semaphore logic from `callLLM` — it no longer needs to acquire/release since WorkerPool handles concurrency:

```go
// callLLM — simplified, WorkerPool handles all scheduling
func (b *Bot) callLLM(ctx context.Context, sessionKey string, messages []openai.ChatCompletionMessage) string {
    slog.Info("calling deepseek", "session", sessionKey)

    llmCtx, llmCancel := context.WithTimeout(ctx, 30*time.Second)
    defer llmCancel()
    reply, err := b.llm.Chat(llmCtx, messages)
    if isTimeout(err) {
        slog.Info("deepseek timeout, retrying once", "session", sessionKey)
        retryCtx, retryCancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer retryCancel()
        reply, err = b.llm.Chat(retryCtx, messages)
    }
    if err != nil {
        slog.Error("deepseek error", "err", err)
        if isTimeout(err) {
            return "抱歉，回复超时，请稍后再试。"
        }
        return "抱歉，我暂时无法回复。"
    }
    return reply
}
```

- [ ] **Step 2: Build and fix compilation**

Run: `cd /Users/Ein/project2/robot && go build ./...`
Expected: compiles or reveals missing pieces — fix iteratively

- [ ] **Step 3: Run full test suite**

Run: `cd /Users/Ein/project2/robot && go test ./...`
Expected: PASS (may need to update bot_test.go)

- [ ] **Step 4: Commit**

```bash
git add internal/core/bot.go
git commit -m "feat: integrate WorkerPool into bot dispatch"
```

---

### Task 6: Update config

**Files:**
- Modify: `internal/core/config.go`
- Modify: `config.example.yaml`

- [ ] **Step 1: Add WorkerConfig to config.go**

```go
type Config struct {
    ...
    Worker WorkerConfig `yaml:"worker"`
}

type WorkerConfig struct {
    Count     int `yaml:"count"`      // worker 数量，默认 3
    QueueSize int `yaml:"queue_size"` // 队列缓冲，默认 64
}
```

In `LoadConfig`, add defaults:
```go
if cfg.Worker.Count == 0 {
    cfg.Worker.Count = 3
}
if cfg.Worker.QueueSize == 0 {
    cfg.Worker.QueueSize = 64
}
```

- [ ] **Step 2: Add to config.example.yaml**

```yaml
worker:
  count: 3
  queue_size: 64
```

- [ ] **Step 3: Build + test**

Run: `cd /Users/Ein/project2/robot && go build ./... && go test ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/core/config.go config.example.yaml
git commit -m "feat: WorkerPool config driven"
```

---

### Task 7: Cleanup — remove llmSem

**Files:**
- Modify: `internal/core/bot.go`

- [ ] **Step 1: Remove llmSem and personaMu (if still unused)**

Remove:
```go
var (
    personaMu sync.Mutex
    llmSem = make(chan struct{}, 3)
)
```

If `personaMu` is still used by `registerPersona`/`updateBinding`, keep it.

- [ ] **Step 2: Build + test**

Run: `cd /Users/Ein/project2/robot && go build ./... && go test ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/bot.go
git commit -m "refactor: remove llmSem, superseded by WorkerPool"
```

---

### Task 8: End-to-end smoke test

**Files:**
- Modify: `internal/core/bot_test.go` (ensure existing tests pass with WorkerPool)

- [ ] **Step 1: Run all tests**

Run: `cd /Users/Ein/project2/robot && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run vet**

Run: `cd /Users/Ein/project2/robot && go vet ./...`
Expected: no output

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "test: full suite passing with WorkerPool integration"
```
