# P4: RAG Knowledge Base

> 2026-06-26 | draft

## Goal

Extend character knowledge beyond SKILL.md by indexing wiki content into a vector database. During conversation, retrieve relevant context and inject it into the prompt for DeepSeek to answer from.

## Architecture

```
/注入知识 <slug> <wiki_url>
    │
    ▼
fetcher.go (reuse) ──▶ prts_parser.py (reuse) ──▶ chunker ──▶ embedder ──▶ Qdrant
                                                                           │
                                                                  tagged {slug}
                                                                           │
用户: "源石技艺是什么"                                                       │
    │                                                                       │
    ▼                                                                       │
embedder(question) ──▶ Qdrant search(top 10, filter slug) ──▶ re-rank ──▶ top 3
                                                                              │
                                                                              ▼
                                                 prompt: system + context + question
                                                                              │
                                                                              ▼
                                                                          DeepSeek
```

## Components

### Chunker — Recursive Split

Priority: `\n\n` → `\n` → `。` → char. Keeps sentences intact.

```go
func RecursiveSplit(text string, maxLen int) []Chunk {
    // Try split by "\n\n", if chunk too long, try "\n", then "。"
    // Each chunk: {Text, Index, Source}
}
```

Target: ~500 chars per chunk, ~10% overlap between adjacent chunks to preserve cross-boundary context.

### Embedder — Zhipu embedding-3

```
POST https://open.bigmodel.cn/api/paas/v4/embeddings
Body: {model: "embedding-3", input: [...]}
Response: {data: [{embedding: [...]}]}
```

1024-dimension vectors. Batch up to 16 texts per request.

```go
type Embedder struct {
    apiKey string
    model  string  // "embedding-3"
}

func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error)
```

### Qdrant — Vector Store

`docker-compose.yml` addition:

```yaml
qdrant:
  image: qdrant/qdrant:v1.16
  ports: ["6333:6333"]
  volumes: ["./qdrant_data:/qdrant/storage"]
```

Collection per character slug, or single collection with `{slug}` payload filter. Single collection is simpler.

```go
type QdrantStore struct {
    client *qdrant.Client
}

func (s *QdrantStore) Upsert(ctx context.Context, slug string, chunks []Chunk, vectors [][]float32) error
func (s *QdrantStore) Search(ctx context.Context, slug string, vector []float32, topK int) ([]ScoredChunk, error)
```

### Retriever — Search + Re-rank + Prompt

```go
func (r *Retriever) Retrieve(ctx context.Context, question string, slug string) (string, error) {
    // 1. Embed question
    qVec := r.embedder.Embed(ctx, []string{question})

    // 2. Qdrant search top 10
    candidates := r.store.Search(ctx, slug, qVec[0], 10)

    // 3. Re-rank: BM25 keyword overlap as secondary score
    //    combined = 0.7 * vector_score + 0.3 * keyword_score
    ranked := rerank(candidates, question)

    // 4. Keep top 3
    top := ranked[:min(3, len(ranked))]

    // 5. Build context string
    return buildContext(top), nil
}

func buildContext(chunks []ScoredChunk) string {
    var b strings.Builder
    for i, c := range chunks {
        fmt.Fprintf(&b, "[资料%d] %s\n\n", i+1, c.Text)
    }
    return b.String()
}
```

### Prompt Injection

```go
// In processMessage / callLLM:
context, _ := b.retriever.Retrieve(ctx, text, persona.Slug)
if context != "" {
    systemPrompt = systemPrompt + "\n\n参考以下资料回答，资料不足以回答时如实告知：\n" + context
}
```

## Commands

```
/注入知识 <slug> <wiki_url>    异步，约 30s
/知识状态 <slug>                返回该角色已注入的文档数、区块数
/清除知识 <slug>                删除该角色所有向量
```

## Files

| File | Action | Lines |
|------|--------|-------|
| `internal/rag/chunker.go` | Create | ~60 |
| `internal/rag/embedder.go` | Create | ~40 |
| `internal/rag/qdrant.go` | Create | ~80 |
| `internal/rag/retriever.go` | Create | ~60 |
| `internal/rag/chunker_test.go` | Create | ~30 |
| `internal/core/bot.go` | Modify | +retriever call in callLLM, +3 commands |
| `docker-compose.yml` | Modify | +qdrant service |
| `go.mod` | Modify | +qdrant-go-client |

Total: ~270 lines Go + 1 Docker service.

## What Doesn't Change

- `fetcher.go` / `prts_parser.py` — reused for ingestion
- `client.go` — DeepSeek call unchanged, context is just part of system prompt
- Persona, session, config, proxy — untouched

## Testing

- [ ] Chunker: wiki article → correct number of chunks, sentences not broken mid-way
- [ ] Embedder: single text → 1024-dim vector returned
- [ ] Qdrant: upsert → search returns correct chunks with scores
- [ ] Retriever: question → relevant context returned, irrelevant question → empty
- [ ] Full flow: /注入知识 → /对话 → answer references context

## Metrics

~270 lines Go. New dependencies: `qdrant-go-client`, Zhipu API key. No Python changes needed.
