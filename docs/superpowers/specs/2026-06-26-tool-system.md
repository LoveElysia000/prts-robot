# P3: Script-Driven Tool System

> 2026-06-26 | draft

## Goal

Let characters call external tools (weather, search, etc.) during conversation. Tools are defined as files (`tools/*.yaml` + optional `tools/*.py`), auto-discovered on reload, filtered per character via `personas.yaml` `skills` field. Architecture keeps an MCP-compatible adapter slot for future migration.

## Design

### Directory Structure

```
tools/
├── weather.yaml       # tool definition
├── weather.py         # handler (optional, defaults to echo)
├── search.yaml
├── search.py
└── ...                # existing Python scripts (prts_parser.py, character_skill_writer.py) untouched
```

### Tool Definition (YAML)

```yaml
# tools/weather.yaml
name: get_weather
description: 查询指定城市的当前天气
parameters:
  type: object
  properties:
    city:
      type: string
      description: 城市名称，如"北京"
  required: [city]
handler: python3 tools/weather.py
timeout: 10s
```

### Handler Script (Python)

```python
# tools/weather.py
import sys, json
args = json.loads(sys.argv[1])
city = args["city"]
# real implementation would call an API
result = f"{city}今天晴天，25°C"
print(json.dumps({"result": result}))
```

Input via stdin or argv as JSON. Output to stdout as JSON `{"result": "..."}`. Exit code 0 = success, non-zero = error (stderr captured).

### Registration

```go
// internal/llm/tool/registry.go

type ToolDef struct {
    Name        string
    Description string
    Parameters  json.RawMessage  // JSON Schema
    Handler     string           // "python3 tools/weather.py"
    Timeout     time.Duration
}

type Registry struct {
    tools map[string]*ToolDef
}

func Load(dir string) (*Registry, error)  // scans dir/*.yaml
func (r *Registry) Filter(names []string) []*ToolDef  // by persona.Skills
func (r *Registry) ToOpenAI(defs []*ToolDef) []openai.Tool  // convert schema
```

`Load` is called on startup and `/角色 重载`. No recompile needed — add a yaml + script, reload.

### Execution Flow

```
User: "北京天气怎么样"  (角色 Lin, skills: [weather])
    │
    ▼
bot → DeepSeek (tools: [get_weather])
    │
    ▼
DeepSeek → tool_call(name="get_weather", arguments={city:"北京"})
    │
    ▼
bot → exec.Command("python3", "tools/weather.py", `{"city":"北京"}`)
    │
    ▼
stdout: {"result":"北京今天晴天，25°C"}
    │
    ▼
bot → DeepSeek (tool result fed back as assistant message)
    │
    ▼
DeepSeek: "北京今天晴天，25°C，适合出门！"
```

Timeouts: tool execution capped at 10s (configurable per yaml). If timeout or error, the error text is fed back to DeepSeek as the tool result.

### MCP Adapter Slot (future)

```go
// internal/llm/tool/source.go

type ToolSource interface {
    List(context.Context) ([]*ToolDef, error)
}

type FileSource struct { dir string }      // current implementation
type MCPSource struct { client *mcp.Client }  // future
```

`Registry` takes `[]ToolSource`. Switching to MCP = add an `MCPSource`, zero changes to execution flow.

### Integration with Bot

In `processMessage` → `callLLM` (or the future WorkerPool handler):

```go
// After building messages, before LLM call:
tools := b.toolRegistry.Filter(persona.Skills)
openaiTools := b.toolRegistry.ToOpenAI(tools)

resp, err := b.llm.Chat(ctx, messages, openaiTools...)
if resp.IsToolCall() {
    result := b.executeTool(ctx, resp.ToolCall)
    messages = append(messages, toolCallMsg, toolResultMsg)
    resp, err = b.llm.Chat(ctx, messages, openaiTools...) // retry once
}
```

Max 1 tool call + 1 follow-up per message (no infinite loops).

### Files

| File | Action |
|------|--------|
| `internal/llm/tool/registry.go` | Create — Load, Filter, ToOpenAI |
| `internal/llm/tool/executor.go` | Create — execute tool via os/exec |
| `internal/llm/tool/registry_test.go` | Create |
| `internal/core/bot.go` | Modify — wire tool call into callLLM |
| `tools/weather.yaml` | Create — example tool |
| `tools/weather.py` | Create — example handler |

### What Doesn't Change

- Persona system (`Persona.Skills` already exists)
- `client.go` (`BuildMessages` already accepts `tools`)
- Session, config, Docker

### Testing

- [ ] Load tools from directory → correct count and schema
- [ ] Filter by skills list → only matching tools returned
- [ ] ToOpenAI conversion → valid openai.Tool structure
- [ ] Execute tool → stdout captured, timeout honored
- [ ] Execute tool timeout → error returned, not hung
- [ ] Full flow: chat → tool_call → execute → result → final reply

### Metrics

~200 lines Go + 2 example files. No new dependencies.
