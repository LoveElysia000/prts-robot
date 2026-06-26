# Worker Pool & Unified Pipeline Design

> **Date:** 2026-06-26
> **Status:** draft

## Goal

Replace the current `goroutine-per-message + llmSem` pattern with a Worker Pool that provides bounded concurrency, priority scheduling, and progress feedback via Discord message editing.

## Scope

Only message handling in `internal/core/bot.go` is refactored. The underlying layers—LLM client, persona manager, session storage, wiki generator, config, Docker—remain untouched.

## Design

### Task Classification

```
消息到达 → handleMessage()
              │
              ├─ /帮助、/角色 列表/切换/重载 → 同步返回（不走 Pool，纯内存/文件操作）
              │
              └─ 聊天消息、/角色校正、/生成角色 → 创建 Task → WorkerPool.Submit()
```

**Not in Pool** (synchronous, immediate return):
- `/帮助` — string concatenation
- `/角色 列表` — in-memory query
- `/角色 切换` — file write + Reload
- `/角色 重载` — file read + Reload

**In Pool** (asynchronous, may block):
- Chat messages — LLM call
- `/角色校正` — LLM call
- `/生成角色` — wiki fetch + LLM ×4

### Priority

| Priority | Tasks | Behavior |
|----------|-------|----------|
| Light(1) | Chat, `/角色校正` | Always dequeued before Heavy tasks |
| Heavy(2) | `/生成角色` | Only processed when no Light tasks are waiting |

Nested `select` in the worker loop (see below) guarantees Light priority. If all 3 workers are busy with Heavy tasks, a new Light task still gets picked up as soon as ANY worker finishes — it never waits behind another Light task.

### Core Types

```go
type TaskPriority int
const (
    PriorityLight TaskPriority = 1  // chat, /角色校正
    PriorityHeavy TaskPriority = 2  // /生成角色
)

type Task struct {
    Priority TaskPriority
    Handler  func(ctx context.Context) (string, error)
    OnStart  func()  // called when worker dequeues, before Handler()
}

type WorkerPool struct {
    lightCh chan *Task
    heavyCh chan *Task
    wg      sync.WaitGroup
    ctx     context.Context
    cancel  context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool
func (p *WorkerPool) Submit(ctx context.Context, task *Task) (string, error)
func (p *WorkerPool) Shutdown()
```

`Submit` **blocks** until the task completes or `ctx` is cancelled. The caller is responsible for running `Submit` in a goroutine (to avoid blocking the Discord event loop) and for managing progress feedback (placeholder message, typing indicator, message edit).

### Priority Queue Implementation

Worker loop using nested `select`:

Nested `select` gives Light priority without starvation: workers always try Light first, fall back to Heavy only when Light is empty. All 3 workers use this same loop.

### Progress Feedback — Ownership

Placeholder message creation happens in `bot.go` **before** calling `Submit()`. Since `Submit` blocks, the caller wraps it in a goroutine:

```go
// bot.go — caller wraps Submit in a goroutine so Discord event loop isn't blocked
go func() {
    msg, _ := s.ChannelMessageSendReply(channelID, "⏳ 排队中...", ref)
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()
    reply, err := pool.Submit(ctx, &Task{
        Priority: PriorityLight,
        Handler:  func(ctx context.Context) (string, error) { ... },
        OnStart:  func() { s.ChannelTyping(channelID) },
    })
    if err != nil {
        s.ChannelMessageEdit(channelID, msg.ID, "抱歉，处理超时，请稍后再试。")
        return
    }
    s.ChannelMessageEdit(channelID, msg.ID, reply)
}()
```

WorkerPool is Discord-agnostic — it only knows about `Handler` and `OnStart` callbacks. All Discord API calls stay in `bot.go`.

### What Doesn't Change

- LLM call flow (`buildMessages` / `callLLM` / retry / timeout)
- Persona system (`manager.go`, `card.go`, `corrector.go`)
- Session storage (`session/manager.go`)
- Wiki generation (`generator/`)
- Configuration, Dockerfile, docker-compose

### Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/core/worker.go` | **Create** | WorkerPool + Task + priority queue (~100 lines) |
| `internal/core/worker_test.go` | **Create** | Unit tests for submission, priority, shutdown |
| `internal/core/bot.go` | **Modify** | Replace goroutine dispatch with WorkerPool.Submit() |
| `internal/core/bot_test.go` | **Modify** | Update tests for new dispatch pattern |

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Task completed | `Submit()` returns `(reply, nil)` |
| `ctx` cancelled before dequeued | `Submit()` returns `("", ctx.Err())` — caller sends timeout message |
| `ctx` cancelled during Handler | Handler returns its own error, `Submit()` returns `("", err)` |
| Task timeout (LLM) | Existing retry + timeout logic in Handler unchanged |
| Worker panic | `recover()` in worker, log error, `Submit()` returns `("", fmt.Errorf("..."))` |
| Shutdown (`SIGINT`) | Deny new submissions, finish in-flight tasks, workers exit |

### Testing

- [ ] WorkerPool: submit light task → processed immediately
- [ ] WorkerPool: submit 2 heavy tasks → 1 worker per task
- [ ] WorkerPool: submit 4 tasks (2L + 2H) → lights complete before heavies
- [ ] WorkerPool: shutdown with pending tasks → graceful drain
- [ ] bot.go: `/帮助` returns synchronously, no pool interaction
- [ ] bot.go: chat message → placeholder sent → edited with reply
- [ ] bot.go: `/生成角色` → placeholder → edited with completion message
