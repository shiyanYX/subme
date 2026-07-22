# SubMe

Proxy subscription management tool that fetches, processes, and serves Clash-format subscription links from proxy providers.

## Language

**Collector**:
A self-contained JS script (run as a subprocess) that knows how to discover a **Provider's** current panel URL, log in, and extract the active **Clash Subscription** URL. Each **Provider** has its own Collector.
_Avoid_: adapter, connector, driver, 插件

**Panel**:
The CMS/control panel software used by a **Provider** to manage subscribers (e.g., SSPanel, V2Board, ProxyPanel).
_Avoid_: 面板, dashboard, backend

**Landing Page**:
A stable URL (e.g. Blogspot, GitHub Page) that redirects to or announces the **Provider's** current **Panel** URL. Used because **Provider** domains change frequently due to censorship.
_Avoid_: 发布页, portal, gateway

**Collectors Directory**:
A directory containing one subdirectory per **Provider**, each holding a `collector.js` and a `config.yaml`.
_Avoid_: modules folder, extensions

**Mode 1 (Link-Only Update)**:
SubMe logs into a **Provider's** **Panel** via the **Collector**, extracts the current **Clash Subscription** URL, and writes it into a **Subscriber's** local Clash configuration file (replacing the `url:` field in `proxy-providers`).
_Avoid_: 模式一, URL mode

**Mode 2 (Cache & Re-Serve)**:
SubMe fetches the **Clash Subscription** via the extracted URL, caches the response (proxies + groups + rules), and re-serves it from a local HTTP endpoint that mirrors the original response exactly.
_Avoid_: 模式二, proxy mode, local cache mode

**Manual Trigger**:
A one-shot `subme update` CLI command that runs all **Collectors** immediately.
_Avoid_: one-time run

**Auto Refresh**:
A built-in scheduler that re-runs all **Collectors** on a configurable interval (per-Provider, per-Collector configurable).
_Avoid_: cron, timer, scheduler (in code, but not in glossary)

**Clash Subscription**:
A URL that, when fetched, returns a YAML document in Clash format containing proxies, proxy-groups, and rules.
_Avoid_: 订阅链接, node list, config

**Provider**:
A proxy service operator (e.g., an "airport"/VPN service) that issues Clash subscription URLs to its subscribers.
_Avoid_: 机场, upstream, vendor

**Subscriber**:
A person or system that consumes one or more Clash subscriptions.
_Avoid_: 用户, client, end-user

## Relationships

- A **Provider** issues **Clash Subscriptions** to **Subscribers**
- A **Subscriber** may collect subscriptions from multiple **Providers**
- A **Clash Subscription** belongs to exactly one **Provider**
- A **Collector** knows how to log into exactly one **Provider's** **Panel** (or discover its current URL via a **Landing Page**)

## Example dialogue

> **Dev:** "When a **Subscriber** adds a **Provider's** subscription URL, do we fetch it immediately or on a schedule?"
> **Domain expert:** "Both — fetch immediately on add, then refresh periodically."

## Flagged ambiguities

- "面板插件" was initially used to mean a shared Collector per Panel type — resolved: each **Provider** gets its own **Collector** directory, even if they use the same **Panel** software.
