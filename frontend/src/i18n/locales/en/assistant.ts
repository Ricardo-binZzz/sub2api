export default {
  assistant: {
    title: 'AI Assistant',
    subtitleUser: 'Answers using only your usage and recent request errors.',
    subtitleAdmin: 'Analyzes site-wide traffic, latency, errors, groups, and channel health.',
    personalContext: 'Personal context',
    operationsContext: 'Operations context',
    readOnly: 'Read-only',
    model: 'Model: {model}',
    welcomeUser: 'Ask about an error, your recent usage, account access, or ways to reduce usage cost.',
    welcomeAdmin: 'Ask about traffic, error rates, latency, slow groups, channel health, or operating priorities.',
    placeholder: 'Ask the assistant...',
    send: 'Send',
    sending: 'Thinking...',
    retry: 'Retry status check',
    unavailableTitle: 'Assistant is not configured',
    unavailableUser: 'Contact the site administrator to enable the assistant.',
    unavailableAdmin: 'Enable the assistant and configure its OpenAI-compatible provider in the server configuration.',
    loadError: 'Could not load assistant status.',
    requestError: 'The assistant is temporarily unavailable. Please try again.',
    privacyUser: 'Only your aggregate usage and recent sanitized errors are sent as context.',
    privacyAdmin: 'Only site operational aggregates are sent as context. The assistant cannot change configuration.',
    disclaimer: 'Suggestions are advisory. The assistant cannot execute repairs, modify balances, or change settings.',
    remaining: '{count} characters remaining'
  }
}
