export default {
  assistant: {
    title: 'AI 助手',
    subtitleUser: '仅根据你的用量和最近请求错误回答问题。',
    subtitleAdmin: '分析全站流量、延迟、错误、分组与渠道健康。',
    personalContext: '个人上下文',
    operationsContext: '运营上下文',
    readOnly: '只读',
    model: '模型：{model}',
    welcomeUser: '可以询问请求报错、近期用量、账号访问问题，或如何降低使用成本。',
    welcomeAdmin: '可以询问流量、错误率、延迟、慢分组、渠道健康或当前运营优先级。',
    placeholder: '向助手提问...',
    send: '发送',
    sending: '分析中...',
    retry: '重新检查状态',
    unavailableTitle: 'AI 助手尚未配置',
    unavailableUser: '请联系站点管理员启用 AI 助手。',
    unavailableAdmin: '请在服务端配置中启用助手，并设置兼容 OpenAI 的模型服务。',
    loadError: '读取助手状态失败。',
    requestError: 'AI 助手暂时不可用，请稍后重试。',
    privacyUser: '上下文仅包含你的汇总用量和已脱敏的近期错误。',
    privacyAdmin: '上下文仅包含站点运营汇总数据，助手不能修改配置。',
    disclaimer: '回答仅供参考。助手不能执行修复、修改余额或变更设置。',
    remaining: '还可输入 {count} 个字符'
  }
}
