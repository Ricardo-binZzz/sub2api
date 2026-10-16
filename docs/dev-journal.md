# Development journal

## 2026-10-16: generation TPS

The usage table, operations TPS distribution and per-model token statistics now
divide output tokens by total duration minus recorded first-token latency.
The table uses raw milliseconds and displays two decimals. Missing or invalid
timings are excluded; image and unsupported request types remain excluded.
Aggregate throughput, billing, stored usage and TTFT instrumentation are unchanged.

The three focused frontend suites passed (72 tests), including the reference
170 / 0.859 = 197.90 tok/s example. Backend integration and production build
results are recorded in /opt/sub2api-deploys/tps-generation-20261016/README.md.
The referenced codex-outbound-identity convention was absent in this checkout.
This change does not touch outbound request identity or account extra fields.

## 2026-10-16: cybersecurity refusal visibility

Incident 626572 returned HTTP 403 with content_policy_violation and the explicit
cybersecurity warning, without error.code. Previously the request retried and
failed over, then became a generic 502. Recognize this precise wrapper using the
shared detector; preserve the warning in HTTP passthrough and buffered SSE JSON
conversion. Existing streaming, compatibility and WebSocket paths share detection.
Regression tests cover recognition boundaries, one-call HTTP forwarding, both
SSE event shapes, buffered and active streams, Chat/Anthropic, WS and metadata
sanitization. Deployment evidence is in
/opt/sub2api-deploys/cyber-policy-message-20261016/README.md.
The referenced outbound identity convention remains absent; this change does not
modify outbound identity or account extra persistence.

## 2026-10-16: yunbay standby

User selected ordinary official relay and Fast groups only. Added a group-scoped
standby gate because advanced score/Top-K selection cannot enforce strict backup
semantics using priority alone. Yunbay activates only when the configured primary
accounts cannot serve the current request; normal recovery overrides session
stickiness while actual continuation ownership is preserved. Existing group 82
behavior and model mappings remain unchanged. The upstream models endpoint
confirmed four GPT models but not codex-auto-review. No unsupported mapping added.
Referenced outbound identity/account-extra-freshness files are absent. Configuration
uses the existing atomic bulk extra merge API, avoiding stale whole-extra writes.
Added the missing unit tag to the previous cyber test fixture that uses unit-only
helpers so default service test compilation also works.
Deployment evidence: /opt/sub2api-deploys/yunbay-standby-20261016/README.md.
