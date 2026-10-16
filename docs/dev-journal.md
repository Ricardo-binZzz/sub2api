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
