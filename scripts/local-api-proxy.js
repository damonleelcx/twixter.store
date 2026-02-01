#!/usr/bin/env node
/**
 * 本地 API 代理：监听 80 端口，将 http://api.twixter.local 的请求转发到 http://127.0.0.1:8080
 * 用于「仅用 port-forward、不用 minikube tunnel」时，让前端（已构建为请求 api.twixter.local）能连上后端。
 *
 * 使用前：
 * 1. hosts 中添加：127.0.0.1 api.twixter.local
 * 2. 运行：kubectl port-forward svc/twixter-backend 8080:8080
 * 3. 本机运行：node scripts/local-api-proxy.js（Windows 上 80 端口通常需管理员权限）
 *
 * 若 80 端口被占用或无权绑定，可设置环境变量：PORT=8081 node scripts/local-api-proxy.js
 * 此时需在 hosts 中写：127.0.0.1 api.twixter.local，且浏览器访问前端时用 http://localhost:3000?api=http://api.twixter.local:8081
 * 但前端已写死 api.twixter.local 且默认 80 端口，故推荐以管理员身份运行本脚本绑定 80。
 */

const http = require("http");

const LISTEN_PORT = parseInt(process.env.PORT || "80", 10);
const TARGET = "http://127.0.0.1:8080";

const server = http.createServer((clientReq, clientRes) => {
  const opts = {
    hostname: "127.0.0.1",
    port: 8080,
    path: clientReq.url,
    method: clientReq.method,
    headers: clientReq.headers,
  };
  const proxyReq = http.request(opts, (proxyRes) => {
    clientRes.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(clientRes);
  });
  proxyReq.on("error", (err) => {
    console.error("Proxy error:", err.message);
    clientRes.writeHead(502, { "Content-Type": "text/plain" });
    clientRes.end("Bad Gateway: " + err.message);
  });
  clientReq.pipe(proxyReq);
});

server.listen(LISTEN_PORT, "127.0.0.1", () => {
  console.log(`Local API proxy: http://127.0.0.1:${LISTEN_PORT} -> ${TARGET}`);
  console.log("Ensure hosts has: 127.0.0.1 api.twixter.local");
  console.log("Keep kubectl port-forward svc/twixter-backend 8080:8080 running.");
});
server.on("error", (err) => {
  if (err.code === "EACCES" && LISTEN_PORT < 1024) {
    console.error("Binding to port %d may require admin/root. On Windows, run as Administrator.", LISTEN_PORT);
  }
  if (err.code === "EADDRINUSE") {
    console.error("Port %d is already in use. Options:", LISTEN_PORT);
    console.error("  1) Find and stop the process: netstat -ano | findstr :%d  (Windows), then taskkill /PID <pid> /F", LISTEN_PORT);
    console.error("  2) Use another port: set PORT=8081 && node scripts/local-api-proxy.js");
    console.error("     Then rebuild frontend with NEXT_PUBLIC_API_URL=http://api.twixter.local:8081 and redeploy.");
  }
  console.error(err);
  process.exit(1);
});
