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

| Priority | Tasks | Reservation |
|----------|-------|-------------|
| Light(1) | Chat, `/角色校正` | At least 1 worker always available |
| Heavy(2) | `/生成角色` | Max 2 workers can process simultaneously |

Workers always pick Light tasks before Heavy tasks. If no Light tasks are queued, all 3 workers can process Heavy tasks.

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
func (p *WorkerPool) Submit(task *Task) (string, error) // blocks until task completes
func (p *WorkerPool) Shutdown()                          // drains queue gracefully
```

### Priority Queue Implementation

Two separate channels with a unified `select` in each worker:

```go
type WorkerPool struct {
    lightCh  chan *Task   // Light tasks (chat, /角色校正)
    heavyCh  chan *Task   // Heavy tasks (/生成角色)
    workers  int
}

// Worker loop
func (p *WorkerPool) run(id int) {
    for {
        select {
        case task := <-p.lightCh:
            task.execute()
        default:
            select {
            case task := <-p.lightCh:
                task.execute()
            case task := <-p.heavyCh:
                task.execute()
            }
        }
    }
}
```

Nested `select` gives Light priority without starvation: workers always try Light first, fall back to Heavy only when Light is empty. All 3 workers use this same loop—no dedicated "light-only" worker needed since the priority mechanism guarantees Light tasks are always picked first.

### Progress Feedback — Ownership

Placeholder message creation happens in `bot.go` **before** calling `Submit()`:

```go
// bot.go — caller responsibility
msg, _ := s.ChannelMessageSendReply(channelID, "⏳ 排队中...", ref)
pool.Submit(&Task{
    Priority: PriorityLight,
    Handler:  func(ctx context.Context) (string, error) { ... },
    OnStart:  func() { s.ChannelTyping(channelID) },
})
s.ChannelMessageEdit(channelID, msg.ID, reply)  // after Submit returns
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
| Pool full (queue buffer exhausted) | Return "当前请求较多，请稍后再试" immediately |
| Task timeout (LLM) | Existing retry + timeout logic unchanged |
| Worker panic | Recover, log error, return "内部错误" to user |
| Shutdown (`SIGINT`) | Drain queue, finish in-flight tasks, reject new submissions |

### Testing

- [ ] WorkerPool: submit light task → processed immediately
- [ ] WorkerPool: submit 2 heavy tasks → 1 worker per task
- [ ] WorkerPool: submit 4 tasks (2L + 2H) → lights complete before heavies
- [ ] WorkerPool: shutdown with pending tasks → graceful drain
- [ ] bot.go: `/帮助` returns synchronously, no pool interaction
- [ ] bot.go: chat message → placeholder sent → edited with reply
- [ ] bot.go: `/生成角色` → placeholder → edited with completion message
