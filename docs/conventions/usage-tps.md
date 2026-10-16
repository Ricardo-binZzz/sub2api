# Per-request output TPS

Usage latency cells display first-token latency, total duration, and TPS. TPS is
`output_tokens * 1000 / (duration_ms - first_token_ms)`, using unrounded stored
milliseconds and two decimal places in the usage table. The admin overview and
per-model token statistics use the same per-request formula and eligible samples.

A sample requires positive output, a nonnegative recorded first-token latency,
and total duration greater than that latency. Missing/invalid samples display
an em dash and do not contribute to rate aggregates. Image, Live, unknown, and
probe records are excluded. A recorded zero latency is valid. The per-model
request and token totals continue to include all matching records.

This estimates speed after the recorded first event. The existing TTFT mode is
not changed; semantic events need not contain visible text, and upstream output
usage can include reasoning tokens. Do not double-count reasoning tokens.
Aggregate system throughput (tokens per wall-clock window) is a different metric
and must not subtract per-request TTFT. No billing or stored usage fields change.
