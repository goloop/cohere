# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.2] - 2026-08-05

### Documentation
- The package documentation describes how `ai.Request.Format` reaches this
  provider, so it is on the first page a reader sees rather than only in the
  reference.

## [0.2.1] - 2026-08-05

Released by mistake: it carries this changelog entry and nothing else. The
documentation it describes landed in 0.2.2.

## [0.2.0] - 2026-08-05

### Added
- `ai.Request.Format` is mapped onto the provider's own `response_format`, so a request
  for JSON is enforced by the provider rather than merely asked for, and
  `ai.Response.JSON` decodes the reply. `Response.Format` reports
  `ai.FormatNative`. Until now only this package's native request type could
  ask for JSON, so callers going through the provider-agnostic interface had
  to strip code fences from the reply by hand.
  `ai.FormatJSONSchema` sends the schema inside that same object, which is the
  shape this provider expects.

### Changed
- Requires `github.com/goloop/ai` v0.4.0.

## [0.1.2] - 2026-07-10

### Documentation
- `DOC.md`/`DOC.UK.md` note that a stream cut off before its `message-end`
  event yields `io.ErrUnexpectedEOF` instead of a silently-completed response.

## [0.1.1] - 2026-07-10

### Fixed
- A stream cut off before the `message-end` event now surfaces
  `io.ErrUnexpectedEOF` (after best-effort salvaging any partial tool call)
  instead of silently reporting a completed response.

### Changed
- Require `goloop/ai` v0.2.0 (500 no longer retried; jittered backoff).

## [0.1.0] - 2026-07-09

Initial release, built on the `github.com/goloop/ai` interface.

### Added
- `Client` implementing `ai.Client`: `Generate` and streaming `Stream` over the
  v2 chat API, with tool use, multimodal image input and system prompts.
- Native `ChatCompletion` and `ChatStream` (typed Server-Sent Events).
- Embeddings (`Embeddings`, `Embed`) and models (`Models`, `GetModel`).
- Rerank (`Rerank`): score documents against a query, ordered most relevant
  first.
- Functional options: `WithBaseURL`, `WithHTTPClient`, `WithTimeout`,
  `WithMaxRetries`, `WithHeader`.
- Retries on 429 and 5xx with backoff; normalized `*ai.APIError` errors.
