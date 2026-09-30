import { describe, expect, it } from 'vitest'
import { buildApiCallbackUrl } from '../oauthCallbackUrl'

describe('buildApiCallbackUrl', () => {
  const callbackPath = '/auth/oauth/linuxdo/callback'

  it.each([
    ['https://api.example.com', 'https://api.example.com/api/v1/auth/oauth/linuxdo/callback'],
    ['https://api.example.com/', 'https://api.example.com/api/v1/auth/oauth/linuxdo/callback'],
    ['https://api.example.com/v1', 'https://api.example.com/api/v1/auth/oauth/linuxdo/callback'],
    ['https://api.example.com/v1/', 'https://api.example.com/api/v1/auth/oauth/linuxdo/callback'],
    ['https://api.example.com/api/v1', 'https://api.example.com/api/v1/auth/oauth/linuxdo/callback'],
  ])('normalizes API base %s', (baseUrl, expected) => {
    expect(buildApiCallbackUrl(baseUrl, callbackPath)).toBe(expected)
  })

  it('preserves a reverse proxy path prefix', () => {
    expect(buildApiCallbackUrl('https://example.com/sub2api/v1', callbackPath)).toBe(
      'https://example.com/sub2api/api/v1/auth/oauth/linuxdo/callback',
    )
  })

  it('accepts a callback path without a leading slash', () => {
    expect(buildApiCallbackUrl('https://api.example.com/v1', 'auth/oauth/oidc/callback')).toBe(
      'https://api.example.com/api/v1/auth/oauth/oidc/callback',
    )
  })
})
