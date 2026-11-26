# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.6] - 2026-11-26

Keep long responses streaming and check completion data.

## [1.1.5] - 2026-10-18

Keep embedding requests unchanged, reject nil requests, and report malformed stream events.

## [1.1.4] - 2026-09-21

Patch release.

### Fixed
- `Stream` no longer hands out the tool calls a cut-off stream left open. A
  stream that ended before `message-end` was already reported as
  `io.ErrUnexpectedEOF`, but only after the open tool calls had been yielded
  with whatever arguments had arrived, so a tool loop that runs the first call
  and stops would act on an action the model never finished and never see the
  error. The error now comes first and no open call is handed out; a call
  that got its `tool-call-end` is still delivered.
- `Stream` reports a tool call whose arguments are not valid JSON as an error
  instead of handing out an `Input` that cannot be decoded.
- `Stream` reports an event that is not valid JSON as an error instead of
  skipping it, which could quietly drop part of the answer.

### Changed
- Depends on `ai` v1.2.1, whose `SSEEvents` bounds the size of one event, so a
  stream that never ends an event cannot make the reader grow without limit.


Patch release.

### Changed
- Depends on `ai` v1.2.0, which adds the `Capabilities.Images` hint. This driver
  does not generate images, so it reports `Images: false` (the default) - no
  behavior change.

## [1.1.1] - 2026-08-11

Patch release.

### Documentation
- The reference covers the driver's `Capabilities` and the translation of
  model-level 400 refusals into `ai.ErrNoHosted`/`ai.ErrNoFormat`, which until
  now lived only in the godoc.

## [1.1.0] - 2026-08-11

Minor release, on `ai` v1.1.0.

### Added
- `Capabilities` describes what this driver can be asked for. It reports no
  hosted capability, which is the same answer `ai.Request.Hosted` already gets
  from this driver, now available before the call instead of only as an error
  after it. A test pins the two together.
- A provider refusal that means "this model cannot produce that format" is now
  translated into `ai.ErrNoFormat`, so an application degrades with one
  `errors.Is` instead of matching English prose in an error message. The
  provider's own `ai.APIError` is wrapped, not replaced, and stays reachable
  with `errors.As`. The rules are deliberately narrow - only a 400, only a
  format the caller asked for, only an error naming that exact field - because
  mistaking a genuinely bad request for a missing feature would retry it
  forever.

## [1.0.0] - 2026-08-11

First stable release, on `ai` v1.0.0.

### Added
- `ai.Request.Hosted` is answered with `ai.ErrNoHosted` before the request
  leaves, because the provider's search is orchestration around tools the caller supplies,
  which `ai.Request.Tools` already expresses; nothing runs on the provider's
  own side.
  That refusal is the documented behavior rather than a gap left in silence:
  an answer produced without the search that was asked for looks exactly like
  one produced with it, so failing loudly is the only way a caller can tell
  them apart. A caller who would rather have the answer anyway asks again
  without `Hosted`.
- Tests pin it, including that nothing is sent to the provider.

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
