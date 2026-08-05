// Websocket URLs go through the same origin as the page (under /ws) so the
// session cookie rides along; the dev server proxies them to the ws service.
export function wsUrl(path: string): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws${path}`
}
