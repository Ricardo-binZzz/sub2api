export default {
  assistant: {
    operations: {
      title: 'AI Operations', subtitle: 'External promotion, messaging integrations, approvals, and policy-bound automation',
      planNow: 'Plan now', policy: 'Execution policy', connections: 'Connections', addConnection: 'Add connection',
      test: 'Test', noConnections: 'No external connections configured', manualTask: 'Create task', chooseConnection: 'Choose connection',
      createTask: 'Create task', tasks: 'Task queue', platform: 'Platform', status: 'Status', content: 'Content', actions: 'Actions',
      approve: 'Approve', run: 'Run', connection: 'Connection', name: 'Connection name', confirmDelete: 'Delete this connection?',
      pending: 'Queued tasks', today: 'Succeeded today', mode: 'Operating mode', autonomous: 'Autonomous', manual: 'Manual',
      enableRuntime: 'Enable operations runtime', enableAutonomous: 'Enable autonomous planning', autoPublish: 'Allow automatic publishing',
      requireApproval: 'Require administrator approval', planningInterval: 'Planning interval (minutes)', publishInterval: 'Minimum publish interval (minutes)',
      dailyBudget: 'Daily external action budget', quietStart: 'Quiet hours start', quietEnd: 'Quiet hours end',
      runs: 'Run history', task: 'Task', result: 'Result', startedAt: 'Started at', noRuns: 'No runs yet',
      actionFailed: 'Action failed', saved: 'Saved', deleted: 'Deleted', testSucceeded: 'Connection test succeeded',
      taskCreated: 'Task created', taskUpdated: 'Task updated', planCreated: 'Planning completed'
    },
    title: 'AI Assistant',
    subtitleUser: 'Answers using only your usage and recent request errors.',
    subtitleAdmin: 'Analyzes site-wide traffic, latency, errors, groups, and channel health.',
    personalContext: 'Personal context',
    operationsContext: 'Operations context',
    model: 'Model: {model}',
    welcomeUser: 'Ask about an error, your recent usage, account access, or ways to reduce usage cost.',
    welcomeAdmin: 'Ask about traffic, error rates, latency, slow groups, channel health, or operating priorities.',
    placeholder: 'Ask the assistant...',
    send: 'Send',
    sending: 'Thinking...',
    clearConversation: 'Clear conversation',
    quickPrompts: {
      userCost: 'Analyze my recent usage cost',
      userErrors: 'Explain my recent request errors',
      userOptimize: 'Suggest usage optimizations',
      adminTraffic: 'Diagnose traffic and error changes',
      adminGroups: 'Find slow or unhealthy groups',
      adminRisk: 'Review current operational risks',
      adminCost: 'Suggest cost optimizations'
    },
    reward: {
      action: 'Claim new-user reward',
      request: 'Can I claim the new-user reward?',
      granted: '{amount} has been added to your balance and is ready to use.',
      alreadyGranted: 'Your new-user reward was already added. The amount was {amount}.',
      notGranted: 'No reward was added this time. You can continue using the service as usual.'
    },
    retry: 'Retry status check',
    unavailableTitle: 'Assistant is not configured',
    unavailableUser: 'Contact the site administrator to enable the assistant.',
    unavailableAdmin: 'Enable the assistant and configure its OpenAI-compatible provider in the server configuration.',
    loadError: 'Could not load assistant status.',
    requestError: 'The assistant is temporarily unavailable. Please try again.',
    remaining: '{count} characters remaining'
  }
}
