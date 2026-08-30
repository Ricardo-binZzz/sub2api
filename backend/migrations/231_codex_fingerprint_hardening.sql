-- Enable the converged Codex fingerprint policy for existing OAuth-like rows
-- and mint fresh per-account seeds once. Explicit "off" remains an opt-out.
-- Rotating here is intentional: a database copy must not carry the source
-- deployment's effective installation identifiers into a new deployment.
UPDATE accounts
SET extra = jsonb_set(
    jsonb_set(
        COALESCE(extra, '{}'::jsonb),
        '{codex_fingerprint_mode}',
        '"session"'::jsonb,
        true
    ),
    '{codex_fingerprint_seed}',
    to_jsonb(gen_random_uuid()::text),
    true
)
WHERE deleted_at IS NULL
  AND platform = 'openai'
  AND type IN ('oauth', 'setup_token')
  AND COALESCE(extra->>'codex_fingerprint_mode', '') <> 'off';
