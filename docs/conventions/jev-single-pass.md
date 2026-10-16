# JEV single-pass admission

Foreground admission uses one 13-question JEV evaluation, with the existing
transport retry policy and category thresholds. It does not synchronously call
a second engine or infer equivalence to OpenAI's internal enforcement.

For the TypeSafe engine only, current client user text is never removed because
it contains a system-reminder marker. Such text and any role/authorization claims
are untrusted data. Existing assistant/tool-ended request boundaries remain
unchanged to avoid repeatedly judging old user messages during agent tool loops.

Only short referential requests receive nearby user/assistant text: at most two
adjacent turns, at most 600 Unicode characters each. System/developer/tool content
is excluded. Context is JSON-escaped, role-labelled and used only to interpret
the current task. It participates in the input hash. No server-side conversation
history is fetched. This is a limited reference heuristic, not full conversation
coverage. Current input retains the existing 12000-character limit; truncation
and images that JEV cannot inspect are explicitly logged as coverage limitations.

JEV rules v2 retain the independent category definitions but add common guidance
against injection, unsupported intent inference and keyword-only judgments.
Thresholds, configured groups, automatic bans, keyword policy and upstream
cybersecurity refusal handling are not altered by this patch.

On a best-effort 1% sample of completed prechecks, an asynchronous worker may
compare the old extraction/rules with the new result at the same JEV endpoint.
Maximum one comparison per 10 seconds and one active shadow call per process;
skip when foreground work is active, queued work exceeds 16, or the task is more
than five seconds old. Shadow timeout is at most one second with zero retries.
Results go only to structured logs, never hashes, violation counters, emails or
bans. The log includes request ID, decision differences, categories, scores and
duration, not customer text. A disagreement is not a ground-truth label.

Operational errors are asynchronously recorded even when ordinary non-hits are
not retained, without a prompt excerpt or enforcement side effects. Existing
fail-open behavior is preserved and reported as an incomplete audit, not a
verified-safe result. Calibrate thresholds only after human-labelled validation.
