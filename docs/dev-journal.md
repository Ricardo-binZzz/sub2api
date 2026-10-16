# Development journal

## 2026-10-16: official v0.2.17 production merge preparation

User authorized merging all deployed customizations with official v0.2.17,
submitting to the owned fork, waiting for successful CI, then blue/green rollout.
The branch has real production 7d05e63de and upstream tag 060d016041 as merge
parents. Preserve all existing migrations and JEV/standby/Astra/TPS/ticket code.
Resolve the four rehearsed conflicts without discarding WS compatibility calls.
Adapt subscription retirement to CPR while retaining API-key model support.
EPUSDT authenticated order-query compatibility must be deployed and verified
before enabling v0.2.17's mandatory upstream notification confirmation.
Do not weaken payment confirmation or manufacture live payments/refunds to test.
Missing outbound-identity/freshness conventions remain absent; no outbound
identity/account-extra policy redesign is part of this merge.


## 2026-10-16: isolated v0.2.17 upgrade assessment

Prepared a content-level three-way rehearsal using exact official v0.2.16,
the full production 7d05e63de aggregate delta and official v0.2.17 tree. Four
text conflicts resolved in the assessment checkout only: retain custom WS
environment projection before new WithConfig normalization, keep payment return
URL hardening plus upstream RawPath clearing, adapt the flattened account filter
test while preserving CPR. No production source/configuration changes or rollout.
Referenced outbound-identity/account-extra conventions are absent; account extra
and outbound identity policy are not redesigned by this assessment.

Real, read-only v0.2.17-style queries against completed gateway orders: AliMPay
200 with matching amount/trade number; EPUSDT 404. The new mandatory upstream
notification confirmation therefore blocks deployment until EPUSDT implements
the authenticated query contract. Do not remove confirmation as a workaround.
Official retirement filtering also needs custom CPR coverage. No new official
migrations/dependencies; retain all already-executed custom migrations unchanged.

## 2026-10-16: JEV single-pass coverage and rule comparison

User requested improving JEV recall without a synchronous second check. Preserve
thresholds, group scope and disabled automatic bans. Add engine-specific current
input extraction that treats client reminder tags as data; bounded contextual
disambiguation only for short references; and explicit coverage/error evidence.
Version common judging guidance independently of unchanged category questions.
Add bounded asynchronous 1% legacy-rule comparisons at the existing JEV provider,
without enforcement or stored prompt text. No new moderation provider is used.
The referenced outbound identity convention remains absent; this change does not
touch outbound generation identity or account extra. Tests/deployment evidence
will be recorded in the task delivery report.

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
