const net = require("net");
const http = require("http");
const https = require("https");
const { URL } = require("url");

function detectProxy() {
  const socks = process.env.SOCKS_PROXY || process.env.socks_proxy || "";
  const httpProxy = process.env.HTTP_PROXY || process.env.http_proxy || "";
  if (socks) return { type: "socks", url: socks };
  if (httpProxy) return { type: "http", url: httpProxy };
  return null;
}

function readBody(r, onDone) {
  const chunks = [];
  r.on("data", c => chunks.push(c));
  r.on("end", () => onDone(Buffer.concat(chunks).toString("utf8")));
}

function doDirect(targetHost, targetPort, method, path, headers, body) {
  const mod = targetPort === 443 ? https : http;
  return new Promise((resolve, reject) => {
    const req = mod.request({
      hostname: targetHost, port: targetPort, path, method, headers,
      timeout: 10000, rejectUnauthorized: false,
    }, (r) => {
      readBody(r, d => { resolve({ status: r.statusCode, body: d, headers: r.headers }); });
    });
    req.on("error", reject);
    req.on("timeout", () => { req.destroy(); reject(new Error("timeout")); });
    if (body) req.write(body);
    req.end();
  });
}

function doSocks(proxyUrl, host, port, method, path, headers, body) {
  const p = new URL(proxyUrl);
  const socksPort = parseInt(p.port) || 1080;
  return new Promise((resolve, reject) => {
    const socket = net.connect(socksPort, p.hostname, () => {
      socket.write(Buffer.from([0x05, 0x01, 0x00]));
    });
    const cleanup = () => { try { socket.destroy(); } catch {} };
    let state = 0, bufs = [];
    function onData(data) {
      bufs.push(data);
      const buf = Buffer.concat(bufs);
      if (buf.length < 2) return;
      if (buf[0] !== 0x05) { cleanup(); reject(new Error("SOCKS5 invalid version")); return; }
      if (state === 0 && buf[1] === 0x00) {
        state = 1;
        bufs = [];
        const hb = Buffer.from(host, "utf8");
        const req = Buffer.alloc(7 + hb.length + 2);
        req[0] = 0x05; req[1] = 0x01; req[2] = 0x00; req[3] = 0x03;
        req[4] = hb.length; hb.copy(req, 5);
        req[5 + hb.length] = (port >> 8) & 0xff;
        req[6 + hb.length] = port & 0xff;
        socket.write(req);
      } else if (state === 1 && buf.length >= 10) {
        if (buf[1] !== 0x00) {
          cleanup(); reject(new Error("SOCKS5 connection rejected: " + buf[1])); return;
        }
        socket.removeListener("data", onData);
        const mod = port === 443 ? https : http;
        const opts = { socket, host, path, method, headers, timeout: 15000 };
        if (port === 443) opts.rejectUnauthorized = false;
        const req = mod.request(opts, (r) => {
          readBody(r, d => { cleanup(); resolve({ status: r.statusCode, body: d, headers: r.headers }); });
        });
        req.on("error", (e) => { cleanup(); reject(e); });
        if (body) req.write(body);
        req.end();
      }
    }
    socket.on("data", onData);
    socket.on("error", (e) => { cleanup(); reject(e); });
    socket.setTimeout(10000, () => { cleanup(); reject(new Error("SOCKS5 timeout")); });
  });
}

function doHTTPProxy(proxyUrl, host, port, method, path, headers, body) {
  const p = new URL(proxyUrl);
  if (port === 443) {
    return new Promise((resolve, reject) => {
      const preq = http.request({
        hostname: p.hostname, port: parseInt(p.port) || 8080,
        method: "CONNECT", path: host + ":" + port, timeout: 10000,
      });
      preq.on("connect", (res, socket) => {
        const req = https.request({ socket, host, path, method, headers, timeout: 10000, rejectUnauthorized: false },
          (r) => { readBody(r, d => resolve({ status: r.statusCode, body: d, headers: r.headers })); });
        req.on("error", reject);
        req.on("timeout", () => { req.destroy(); reject(new Error("inner timeout")); });
        if (body) req.write(body);
        req.end();
      });
      preq.on("error", reject);
      preq.on("timeout", () => { preq.destroy(); reject(new Error("connect timeout")); });
      preq.end();
    });
  }
  headers["Proxy-Connection"] = "Keep-Alive";
  const absURL = "http://" + host + ":" + port + path;
  return doDirect(p.hostname, parseInt(p.port) || 8080, method, absURL, headers, body);
}

function request(method, host, port, path, body, authToken, useProxy, customHeaders) {
  const proxy = (useProxy !== false) ? detectProxy() : null;
  const headers = {
    "User-Agent": "SubMe/1.0",
    "Accept": "application/json",
    "Content-Type": "application/json",
    ...(authToken ? { "Authorization": authToken } : {}),
    ...(customHeaders || {}),
  };
  if (body) headers["Content-Length"] = Buffer.byteLength(body);

  if (!proxy) return doDirect(host, port, method, path, headers, body);
  if (proxy.type === "socks") return doSocks(proxy.url, host, port, method, path, headers, body);
  return doHTTPProxy(proxy.url, host, port, method, path, headers, body);
}

function apiGET(host, path, token, useProxy, customHeaders) {
  return request("GET", host, 443, path, null, token, useProxy, customHeaders);
}

function apiPOST(host, path, body, token, useProxy, customHeaders) {
  return request("POST", host, 443, path, body, token, useProxy, customHeaders);
}

function requestFollowRedirect(method, host, port, path, body, authToken, useProxy, redirects, customHeaders) {
  redirects = redirects || 0;
  if (redirects > 5) return Promise.reject(new Error("too many redirects"));
  return request(method, host, port, path, body, authToken, useProxy, customHeaders).then(r => {
    if (r.status >= 300 && r.status < 400 && r.headers && r.headers.location) {
      const loc = r.headers.location;
      const u = loc.startsWith("http") ? new URL(loc) : new URL(loc, "https://" + host);
      console.error("[debug] redirect:", r.status, "->", u.href);
      return requestFollowRedirect(method, u.hostname, parseInt(u.port) || 443, u.pathname + u.search, body, authToken, useProxy, redirects + 1, customHeaders);
    }
    return r;
  });
}

function parseYamlish(text) {
  const obj = {};
  for (const line of text.split("\n")) {
    const m = line.match(/^\s*(\w+)\s*:\s*(.*?)\s*$/);
    if (m) obj[m[1]] = m[2].replace(/^["']|["']$/g, "");
  }
  return obj;
}

module.exports = { request, apiGET, apiPOST, requestFollowRedirect, parseYamlish, detectProxy };
