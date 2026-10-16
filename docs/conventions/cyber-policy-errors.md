# Cyber policy errors

Recognize explicit error.code=cyber_policy, plus structured error.type or code
content_policy_violation ONLY when that same error object's message begins with
"This content was flagged for possible cybersecurity risk." (case insensitive).
Inspect error and response.error independently; never scan echoed request input
or classify generic content policy, authentication, or rate-limit messages as cyber.

Recognized refusals use existing request-scoped cyber handling: preserve the
provider message and refusal, do not retry/switch accounts or penalize credentials.
HTTP passthrough emits a minimal JSON error with the original HTTP status; buffered
upstream SSE emits HTTP 403 with the same message and cyber_policy code. Active
streams retain protocol-appropriate errors. Preserve usage evidence and existing
monitoring/session-block policies. Do not turn failures into successful text.
