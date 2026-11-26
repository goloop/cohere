# cohere - довідник

Повний довідник пакета `cohere`: клієнт, спільна модель `goloop/ai`, чат
(інтерфейс і нативний), стрімінг, embeddings і моделі.

Англійська версія: **[DOC.md](DOC.md)**.

## Зміст

- [Ментальна модель](#ментальна-модель)
- [Створення клієнта](#створення-клієнта)
- [Generate і Stream](#generate-і-stream)
- [Структурований вивід](#структурований-вивід)
- [Пошук на боці провайдера](#пошук-на-боці-провайдера)
- [Здатності й відмови на рівні моделі](#здатності-й-відмови-на-рівні-моделі)
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
`io.ErrUnexpectedEOF`, а не тихо повідомляє про завершену відповідь. Виклик
інструмента, який стрім лишив незавершеним, не видається, бо його аргументи теж
могли обірватися; той, що отримав свій `tool-call-end`, видається. Подія, що не
є коректним JSON, або некоректні JSON-аргументи інструмента так само завершують
стрім помилкою.

## Структурований вивід

`ai.Request.Format` лягає на власний `response_format` провайдера, тож запит на JSON
провайдер **дотримує**, а не просто «чує»:

```go
resp, err := c.Generate(ctx, &ai.Request{
	Model:    "the-model",
	Messages: []ai.Message{ai.UserText("Склади SEO-поля для цієї статті.")},
	Format: &ai.Format{
		Type:   ai.FormatJSONSchema,
		Name:   "seo",
		Schema: schema,
	},
})

var seo SEO
err = resp.JSON(&seo)
```

`ai.FormatJSON` іде як `{"type":"json_object"}`; `ai.FormatJSONSchema` кладе
схему в той самий обʼєкт, під ключем `json_schema` - саме такої форми чекає цей
провайдер.


`ai.Response.Format` дорівнює `ai.FormatNative`: цей провайдер дотримує кожну
форму, яку приймає. Які моделі підтримують схемний режим - справа провайдера;
таблиці можливостей тут немає, тож про непідтримувану пару скаже він сам.

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

## Пошук на боці провайдера

`ai.Request.Hosted` отримує `ai.ErrNoHosted` ще до відправлення запиту.

Пошук у цього провайдера - це оркестрація навколо інструментів, які ви даєте
самі, а це вже виражає `ai.Request.Tools`. На боці провайдера не виконується
нічого, тож покласти на це `Hosted` означало б повернути `ToolUse`, на який ви
маєте відповісти, - саме те, чого `Hosted` обіцяє не робити ніколи.

Ця відмова - задокументована поведінка, а не мовчазна прогалина: відповідь, яку
дали без замовленого пошуку, виглядає точно так само, як та, що з пошуком, тож
голосна помилка - єдиний спосіб їх розрізнити. Якщо відповідь усе одно потрібна,
повторіть запит без `Hosted`.

## Здатності й відмови на рівні моделі

Драйвер реалізує `ai.Capable` і звітує відсутність hosted-здатностей - ту саму
відповідь, яку `ai.Request.Hosted` тут і так отримує, тепер доступну до
виклику, а не лише помилкою після. Conformance-тест пінить збіг цих двох.

Відмова, яку провайдер звітує лише 400-кою («ця модель не видасть такий
формат»), тепер приходить загорнутою в `ai.ErrNoFormat`, тож один `errors.Is`
заміняє матчинг англійської прози в тексті помилки; власний `ai.APIError`
провайдера лишається досяжним через `errors.As`. Загортання свідомо вузьке:
лише 400, лише формат, про який запит справді просив, лише помилка, що називає
рівно це поле.

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

## Реліз v1.1.5

Embeddings не змінює переданий запит. Nil-запити chat, stream та embeddings повертають ai.ErrNoRequest. ChatStream завершується з помилкою на пошкодженому JSON.

## Поведінка запиту

`Rerank` вимагає запит. Для пошуку через `Embeddings` задайте `search_query` для запитів і `search_document` для документів.
