export default {
  assistant: {
    operations: {
      title: 'AI 运营', subtitle: '统一管理外部宣传、即时通讯、审批与受策略约束的自主运营',
      planNow: '立即规划', policy: '执行策略', connections: '平台连接', addConnection: '添加连接',
      test: '测试', noConnections: '尚未配置外部平台连接', manualTask: '创建任务', chooseConnection: '选择连接',
      createTask: '创建任务', tasks: '任务队列', platform: '平台', status: '状态', content: '内容', actions: '操作',
      approve: '批准', run: '执行', connection: '平台连接', name: '连接名称', confirmDelete: '确定删除这个连接？',
      pending: '队列任务', today: '今日成功', mode: '运行模式', autonomous: '自主运营', manual: '人工控制',
      enableRuntime: '启用运营运行时', enableAutonomous: '启用自主规划', autoPublish: '允许自动发布',
      requireApproval: '要求管理员审批', planningInterval: '规划间隔（分钟）', publishInterval: '最小发布间隔（分钟）',
      dailyBudget: '每日外部动作上限', quietStart: '静默开始小时', quietEnd: '静默结束小时',
      runs: '运行历史', task: '任务', result: '结果', startedAt: '开始时间', noRuns: '暂无运行记录',
      actionFailed: '操作失败', saved: '已保存', deleted: '已删除', testSucceeded: '连接测试成功',
      taskCreated: '任务已创建', taskUpdated: '任务已更新', planCreated: '规划已完成'
    },
    title: 'AI 助手',
    subtitleUser: '仅根据你的用量和最近请求错误回答问题。',
    subtitleAdmin: '分析全站流量、延迟、错误、分组与渠道健康。',
    personalContext: '个人上下文',
    operationsContext: '运营上下文',
    model: '模型：{model}',
    welcomeUser: '可以询问请求报错、近期用量、账号访问问题，或如何降低使用成本。',
    welcomeAdmin: '可以询问流量、错误率、延迟、慢分组、渠道健康或当前运营优先级。',
    placeholder: '向助手提问...',
    send: '发送',
    sending: '分析中...',
    clearConversation: '清空会话',
    quickPrompts: {
      userCost: '分析我最近的使用成本',
      userErrors: '解释我最近的请求错误',
      userOptimize: '建议如何优化用量',
      adminTraffic: '诊断流量和错误变化',
      adminGroups: '找出缓慢或异常的分组',
      adminRisk: '检查当前运营风险',
      adminCost: '给出成本优化建议'
    },
    reward: {
      action: '领取新用户奖励',
      request: '帮我看看能不能领取新用户奖励',
      granted: '可以，已为你发放 {amount} 余额，刷新后即可使用。',
      alreadyGranted: '这份新用户奖励已经发放过了，金额是 {amount}。',
      notGranted: '这次没有发放奖励，你可以继续正常使用本站服务。'
    },
    retry: '重新检查状态',
    unavailableTitle: 'AI 助手尚未配置',
    unavailableUser: '请联系站点管理员启用 AI 助手。',
    unavailableAdmin: '请在服务端配置中启用助手，并设置兼容 OpenAI 的模型服务。',
    loadError: '读取助手状态失败。',
    requestError: 'AI 助手暂时不可用，请稍后重试。',
    remaining: '还可输入 {count} 个字符'
  }
}
