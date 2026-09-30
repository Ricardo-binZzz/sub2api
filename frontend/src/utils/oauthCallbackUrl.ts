export function buildApiCallbackUrl(baseUrl: string, callbackPath: string): string {
  const base = baseUrl
    .trim()
    .replace(/\/+$/, '')
    .replace(/\/(?:api\/)?v1$/, '')
  const path = callbackPath.startsWith('/') ? callbackPath : `/${callbackPath}`

  return `${base}/api/v1${path}`
}
