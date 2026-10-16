# Group-scoped OpenAI standby

Account extra.openai_standby_for_groups maps group IDs to explicit primary account
ID arrays. The standby remains ordinary in unconfigured groups. The account must
still belong to the target group and pass all normal model/capability/health gates.
A matching standby is excluded whenever a non-excluded primary can serve the
request. Unhealthy/paused/model-ineligible primaries and request-local failures
allow the backup. Priority or session stickiness never overrides this condition.
Actual previous-response owners and guardian parents retain stateful continuity.
Unknown primary lookup state and invalid/self-referential/chained rules fail closed.
Both advanced and legacy schedulers, including initial filtering and DB rechecks,
apply the gate. No per-request policy writes, network probes, or global scheduler
weight changes are required. Admin bulk update atomically merges extra keys and
uses the normal account/group cache invalidation path.

Production: yunbay (286), groups 99 and 104, primaries 272/277/306; all share the
103.41.177.50:8888 upstream at configuration time. These are explicit account IDs,
not DNS matching on every request. Reconfigure the ID arrays if primaries change.
Group 82 membership retained. Group 106 and other primary memberships unchanged.
