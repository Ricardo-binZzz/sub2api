# CPR subscription model retirement

Retirement checks follow the upstream backend, not the credential used to call
this gateway. CPR accounts relay to the ChatGPT Codex backend, so they share
OAuth/setup-token subscription model retirement restrictions. Preserve credential
shadow handling. Ordinary OpenAI API-key accounts remain unaffected.

Apply retirement to the actual one-hop mapping target. Administrator aliases
that explicitly map an old public name to a supported replacement remain valid;
aliases that target a retired model remain unavailable. Scheduling, plain and
Codex model catalogs, and compact-specific mappings/fallback must agree.

No account or group model mapping is automatically rewritten. Stale subscription
entries may be hidden or rejected while valid API-key availability remains visible.
Review explicit replacement mappings separately against upstream capabilities.
