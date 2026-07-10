# cohere - довідник

Повний довідник пакета `cohere`: клієнт, спільна модель `goloop/ai`, чат
(інтерфейс і нативний), стрімінг, embeddings і моделі.

Англійська версія: **[DOC.md](DOC.md)**.

## Зміст

- [Ментальна модель](#ментальна-модель)
- [Створення клієнта](#створення-клієнта)
- [Generate і Stream](#generate-і-stream)
- [Нативний чат](#нативний-чат)
- [Інструменти, зображення й system-промпти](#інструменти-зображення-й-system-промпти)
- [Embeddings](#embeddings)
- [Моделі](#моделі)
- [Опції та помилки](#опції-та-помилки)

## Ментальна модель

`cohere.Client` реалізує `ai.Client` - провайдер-незалежний контракт із
`github.com/goloop/ai`. Спільні `Generate` і `Stream` покривають спільну основу
(чат із інструментами, зображеннями й стрімінгом), тож код проти інтерфейсу
працює з будь-яким провайдером.

Нативна частина - v2 chat API (`ChatCompletion`/`ChatStream`), embeddings і
перелік моделей.

```go
import (
	"github.com/goloop/ai"
	"github.com/goloop/cohere"
)
```

## Створення клієнта

```go
c := cohere.New(os.Getenv("COHERE_API_KEY"))

c = cohere.New(apiKey, cohere.WithTimeout(30*time.Second))
```

Base URL за замовчуванням `https://api.cohere.com`.

## Generate і Stream

```go
resp, err := c.Generate(ctx, &ai.Request{
	Model:    cohere.ModelCommandA,
	System:   "You are concise.",
	Messages: []ai.Message{ai.UserText("Name three primary colors.")},
})
resp.Text()
resp.ToolCalls()
resp.Usage
```

`Stream` повертає `iter.Seq2[ai.Chunk, error]`: текстові дельти чанками з `Text`,
завершений виклик інструмента - чанком із `ToolCall`, фінальний чанк - `Done` і
`Usage`.

```go
for chunk, err := range c.Stream(ctx, req) {
	if err != nil {
		return err
	}
	fmt.Print(chunk.Text)
}
```

Якщо стрім обірвано до події `message-end`, `Stream` віддає
`io.ErrUnexpectedEOF` (після салважу часткового виклику інструмента), а не тихо
повідомляє про завершену відповідь.

## Нативний чат

Для опцій, специфічних для провайдера, будуйте `ChatRequest` і викликайте
`ChatCompletion` чи `ChatStream`. Cohere стрімить типізовані Server-Sent Events
(`content-delta`, `tool-call-start`, `message-end`, ...); нативний `ChatStream`
віддає їх як значення `StreamEvent`.

```go
resp, err := c.ChatCompletion(ctx, &cohere.ChatRequest{
	Model:          cohere.ModelCommandA,
	Messages:       []cohere.ChatMessage{{Role: "user", Content: "as JSON"}},
	ResponseFormat: json.RawMessage(`{"type":"json_object"}`),
})
```

Поле `P` мапиться на nucleus-семплінг Cohere; `MaxTokens` і `StopSequences` - на
`max_tokens` і `stop_sequences`.

## Інструменти, зображення й system-промпти

Інструменти, зображення й system-промпти використовують спільні типи `ai`:
`ai.Tool`, `ai.Image`, `ai.ToolResult` і повідомлення `RoleSystem` або поле
`System`. Результати інструментів надсилаються назад повідомленнями `RoleTool`,
де `ai.ToolResult.ID` збігається з `ai.ToolUse.ID`. Вбудовані байти зображення
надсилаються як base64 data URI (потрібна vision-модель).

## Embeddings

```go
docs, err := c.Embed(ctx, cohere.ModelEmbedV3, "hello", "world") // документи
q, err := c.Embeddings(ctx, &cohere.EmbedRequest{
	Model: cohere.ModelEmbedV3, Texts: []string{"hello"}, InputType: "search_query",
})
```

## Моделі

```go
models, err := c.Models(ctx)
m, err := c.GetModel(ctx, cohere.ModelCommandA)
m.Endpoints // напр. ["chat"]
```

## Опції та помилки

Опції: `WithBaseURL`, `WithHTTPClient`, `WithTimeout`, `WithMaxRetries`,
`WithHeader`.

Невдала відповідь стає `*ai.APIError` зі `Status`, `Message` і сирим тілом:

```go
var apiErr *ai.APIError
if errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests {
	// backoff
}
```

Запити без моделі чи повідомлень падають до мережі з `ai.ErrNoModel` або
`ai.ErrNoMessages`.
