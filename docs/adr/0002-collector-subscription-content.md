# ADR-0002: Collector-provided subscription content

Some collectors require authenticated API calls to obtain a usable subscription URL — they log in, receive an auth token, and must call an "unlock" endpoint before the subscribe URL becomes accessible. The Go server's `fetchSubscription` is a plain HTTP GET with no auth context, so fetching that URL directly returns 403.

## Decision

Collectors may return the actual subscription YAML content in their JSON output via an optional `subscription_content` field. When present, the Go server uses it directly and skips the separate `fetchSubscription` request.

Field added to `collectorOutput`:

```json
{
  "success": true,
  "subscription_url": "https://...",
  "subscription_content": "proxies:\n  - ..."
}
```

The Go server logic:

```
if subscription_content is not empty →
    use it as the subscription YAML
else →
    fetch subscription_url via HTTP (existing behavior)
```

## Consequences

- **Zero impact on existing collectors**: `subscription_content` is optional. Collectors that don't return it work exactly as before.
- **Redundant fetch avoided**: Authenticated collectors don't make a second unauthenticated request that would fail.
- **Collector owns auth context**: The Node.js subprocess already has the auth token/cookies from its login flow, so it can pass the content back naturally.
- **No auth leakage**: The server never needs to know about per-collector auth tokens.

## Files changed

- `internal/collector/collector.go` — added `SubscriptionContent []byte` to `Result` and `collectorOutput`.
- `internal/server/server.go` — checks `len(result.SubscriptionContent) > 0` before calling `fetchSubscription`.
- `collectors/*/collector.js` — collectors may fetch subscribe URL content with auth header and include it in output.
