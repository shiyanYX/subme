# ADR-0001: Collector subprocess architecture

Collectors run as separate Node.js subprocesses rather than as embedded JS in the Go binary. SubMe passes a config file path via argv and receives JSON on stdout. Go manages lifecycle (spawn, timeout, kill) and captures stderr for debug logs.

## Considered Options

- **Goja/QuickJS embedded in Go**: Single binary, no Node.js dependency. But limited npm support — axios/cheerio/puppeteer unavailable — and debugging is harder.
- **WASM plugin**: Cross-language but no standard HTTP session or DOM API support.
- **Node.js subprocess (chosen)**: Full npm ecosystem, easy to debug (`node collector.js` runs standalone), simple IPC contract. The tradeoff: Docker image needs Node.js (30MB overhead on node:alpine).
