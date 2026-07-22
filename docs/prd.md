# SubMe PRD

## Problem Statement

Proxy service Providers frequently change their domain names due to censorship, making Clash subscription URLs unstable. Users must manually log into each Provider's panel to find the current URL and update their Clash configuration. This is repetitive, error-prone, and requires tracking multiple Provider websites with different login procedures and layouts.

Subscribers lack a centralized way to:
- Automatically discover each Provider's current panel URL via stable Landing Pages
- Log into different Provider panels with site-specific procedures (each Provider has its own Collector)
- Cache subscription responses locally to avoid depending on Provider URL stability
- Serve stable local endpoints that Clash can consume via proxy-providers

## Solution

SubMe is a self-hosted Docker service that:
1. Runs per-Provider JS **Collectors** that know how to log into specific Provider panels and extract the current Clash subscription URL
2. Fetches the full subscription YAML via that URL and caches it on disk
3. Serves the cached subscription from a local HTTP endpoint (`/sub/:name`) that Clash proxy-providers consume
4. Refreshes subscriptions on a configurable schedule and via manual trigger
5. Provides a Web UI for managing Providers, viewing subscription links, checking real-time logs, and configuring system settings
6. Notifies the user via WxPusher when collection or refresh operations fail

## User Stories

1. As a Subscriber, I want to add a new Provider with its panel URL or landing page URL and my login credentials, so that SubMe can start collecting its subscription.
2. As a Subscriber, I want to test the connection before saving a new Provider configuration, so that I know my credentials are correct (test failure does not block saving).
3. As a Subscriber, I want SubMe to automatically refresh all cached subscriptions on a configurable schedule, so that I always have up-to-date proxy nodes.
4. As a Subscriber, I want to override the refresh interval per-Provider, so that some Providers can refresh more frequently than the global default.
5. As a Subscriber, I want to manually trigger a refresh for a single Provider or all Providers, so that I can get fresh nodes on demand.
6. As a Subscriber, I want each cached subscription served as a local HTTP endpoint, so that Clash proxy-providers can consume it via `url:`.
7. As a Subscriber, I want SubMe to automatically discover a Provider's new panel URL via its Landing Page, so that I don't need to manually track domain changes.
8. As a Subscriber, I want SubMe to retry with an optional global proxy when the panel URL fails, so that I can still reach censored panels.
9. As a Subscriber, I want automatic fallback: try panel_url first, then with proxy, then discover via landing_page, so that the system is resilient.
10. As a Subscriber, I want to receive WxPusher notifications when a collection or refresh fails, so that I know something is wrong.
11. As a Subscriber, I want to view the status of all Providers on a dashboard, so that I can see which subscriptions are healthy at a glance.
12. As a Subscriber, I want to see real-time logs in the Web UI, so that I can debug collection failures interactively.
13. As a Subscriber, I want to copy each local subscription URL with one click, so that I can paste it into my Clash configuration.
14. As a Subscriber, I want SubMe to update config.yaml with a newly discovered panel_url from the Landing Page, so that future collections skip the discovery step.
15. As a Subscriber, I want to set up an initial admin account on first visit, so that the Web UI is secured.
16. As a Subscriber, I want to configure a global HTTP/SOCKS5 proxy for Collectors, so that I can access panels behind censorship.
17. As a Subscriber, I want to run SubMe as a single Docker container, so that deployment is simple.
18. As a Subscriber, I want SubMe to run all Collectors in parallel when refreshing, so that refresh completes quickly.

## Implementation Decisions

### Architecture

- **Language**: Go for the main binary; Node.js for Collectors (subprocess).
- **Deployment**: Single Docker container based on `node:alpine`, embedding the Go binary.
- **Database**: SQLite for Provider configurations, user accounts, and system settings.
- **Cache**: On-disk YAML files in `/app/cache/{clash_name}.yaml`.
- **Web UI**: React + shadcn/ui, compiled and embedded into the Go binary via `embed.FS`.
- **Collector execution**: Parallel subprocess execution with independent 30-second timeout per Collector.

### Collector Contract

**Invocation**: `node collector.js <path-to-config.yaml>`

**config.yaml**:
```yaml
clash_name: fieniao-jichang
interval: 3600
panel_url: https://xxx.xyz
landing_page: https://xxx.blogspot.com
username: user@example.com
password: xxx
```

**Collector behavior**:
1. Has `panel_url`? → try direct login → success? return
2. Direct login failed and global `proxy` configured? → retry via HTTP_PROXY → success? return
3. Has `landing_page`? → fetch (also uses proxy if configured) to discover new panel_url → goto step 1
4. All failed → return error

**stdout JSON (success)**:
```json
{
  "success": true,
  "panel_url": "https://current-panel-used.xyz",
  "subscription_url": "https://xxx/sub?token=...",
  "update_config": { "panel_url": "https://newly-discovered-panel.xyz" }
}
```
`update_config` is optional; when present, SubMe writes the keys into `config.yaml`.

**stdout JSON (failure)**:
```json
{
  "success": false,
  "error": "Login failed: invalid credentials"
}
```

**stderr**: Free-form debug logging, captured by SubMe for log display and storage.

**Timeout**: SubMe kills the process after 30 seconds. Collector should set its own internal timeouts.

### Modules

- **cmd/subme/main.go**: Entry point. CLI commands: `serve` (HTTP server + scheduler), `update` (one-shot refresh).
- **internal/server/**: HTTP server. Serves Web UI (embedded React), REST API, subscription endpoints (`/sub/:name`), SSE for real-time logs.
- **internal/collector/**: **Deep module.** Runs a Collector subprocess, manages timeout, injects HTTP_PROXY from global config, handles stdout/stderr parsing, writes `update_config` back to `config.yaml`. Interface: `Run(ProviderConfig) → Result`.
- **internal/cache/**: **Deep module.** File-based cache manager. Reads/writes YAML files. Tracks freshness based on interval. Interface: `Get(clashName) → *CacheEntry`, `Set(clashName, yaml)`, `IsFresh(clashName, interval) → bool`.
- **internal/db/**: SQLite layer. Stores Provider configs, admin user, system settings, notification config.
- **internal/scheduler/**: Timer-based auto refresh. Uses global default interval + per-Provider overrides. Triggers parallel Collector execution.
- **internal/notify/**: **Deep module.** WxPusher integration. Interface: `Send(Event) → error`. Uses global app_token and uids from system settings.

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/sub/{clash_name}` | Return cached Clash YAML |
| GET | `/api/providers` | List all providers |
| POST | `/api/providers` | Add a provider |
| PUT | `/api/providers/{id}` | Update a provider |
| DELETE | `/api/providers/{id}` | Delete a provider |
| POST | `/api/providers/{id}/test` | Test connection (run collector, don't save) |
| POST | `/api/providers/{id}/refresh` | Trigger refresh for one provider |
| POST | `/api/refresh` | Trigger refresh for all providers |
| GET | `/api/logs` | SSE endpoint for real-time logs |
| POST | `/api/register` | Register initial admin account |
| POST | `/api/login` | Login |
| GET | `/api/settings` | Get system settings |
| PUT | `/api/settings` | Update system settings |

### System Settings

```yaml
port: 9090
refresh_interval: 3600
proxy: ""
wxpusher:
  app_token: ""
  uids: []
notify_on:
  collect_failure: true
  refresh_failure: true
```

### Notifications

- WxPusher triggers on `collect_failure` and `refresh_failure` (both default on).
- No deduplication: every failure sends a notification.

### Startup Behavior

1. No admin user exists? → start HTTP server immediately, redirect to registration page
2. Admin exists? → parse all Collector configs, run all Collectors **in parallel**, wait for all to complete
3. Start HTTP server with cached subscription endpoints ready
4. Scheduler starts after HTTP server is up

## Testing Decisions

Three modules will have targeted tests:

### `internal/collector/`
- Use mock `.js` scripts to simulate Collector responses (success, failure, timeout, invalid JSON)
- Test: `Run` returns correct `Result` struct for each scenario
- Test: timeout enforcement kills the subprocess after 30s
- Test: `HTTP_PROXY` env var is injected when global proxy is configured
- Test: `update_config` keys are written back to config.yaml on disk
- Prior art: standard Go `testing` + `os/exec` subprocess tests

### `internal/cache/`
- Use temp directories for test cache files
- Test: Set + Get round-trip preserves YAML content exactly
- Test: IsFresh returns correct value based on interval and last-write time
- Test: Get returns nil for missing cache entry
- Prior art: standard Go `testing` with temp file fixtures, table-driven tests

### `internal/notify/`
- Mock HTTP server to simulate WxPusher API
- Test: `Send` makes correct POST request with app_token and content
- Test: `Send` returns error on non-200 response
- Test: `Send` returns error on network failure
- Prior art: `httptest.NewServer` for mocking external APIs

## Out of Scope

- **Mode 1 (Link-Only Update)**: Writing extracted URLs into a Clash config file. Deferred.
- **Plugin/extensibility system**: Adding third-party frontend components. Deferred.
- **Multiple users**: Single admin user only.
- **Non-Clash output formats**: sing-box, Surge, Loon not planned.
- **Node speed testing / traffic monitoring**: Not part of core.
- **Clash configuration management**: SubMe does not manage Clash itself.

## Further Notes

- SubMe assumes Clash is deployed separately and consumes subscription URLs from SubMe's HTTP endpoint.
- Each Provider gets its own Collector directory, even if they use the same Panel software. Code duplication is accepted.
- The `collectors/` directory is mounted as a Docker volume, allowing users to add/modify Collectors without rebuilding the image.
- First-time setup: user visits Web UI, registers admin account, adds Providers, copies subscription URLs into Clash config.
- Three deep modules (collector, cache, notify) are extracted behind simple interfaces for testability.
