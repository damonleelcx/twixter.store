import { NextRequest, NextResponse } from "next/server";

function getBackendUrl(): string {
  const raw =
    process.env.BACKEND_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080";
  return raw.replace(/\/+$/, "");
}

/**
 * 运行时 API 代理：请求时读取 BACKEND_URL，避免 next.config rewrites 在构建时写死目标导致生产环境 ECONNREFUSED。
 * 更具体的路由（如 /api/content/videos/upload）会优先匹配，不会走此代理。
 */
async function proxy(request: NextRequest, pathSegments: string[]) {
  const backendUrl = getBackendUrl();
  if (process.env.NODE_ENV === "production" && /^https?:\/\/localhost(:\d+)?(\/|$)/i.test(backendUrl)) {
    return NextResponse.json(
      {
        error: "Backend not configured",
        details: "Set BACKEND_URL in frontend ConfigMap (e.g. http://twixter-backend:8080) and ensure twixter-frontend-config is applied.",
      },
      { status: 502 }
    );
  }
  const path = pathSegments.join("/");
  const search = request.nextUrl.search;
  const url = `${backendUrl}/api/${path}${search}`;

  const headers = new Headers();
  request.headers.forEach((value, key) => {
    const lower = key.toLowerCase();
    if (lower === "host" || lower === "connection") return;
    headers.set(key, value);
  });

  const init: RequestInit = {
    method: request.method,
    headers,
    cache: "no-store",
  };
  if (request.body && ["POST", "PUT", "PATCH"].includes(request.method)) {
    init.body = request.body;
    (init as RequestInit & { duplex?: "half" }).duplex = "half";
  }

  try {
    const res = await fetch(url, init);
    const contentType = res.headers.get("content-type");
    const body = await res.arrayBuffer();
    return new NextResponse(body, {
      status: res.status,
      statusText: res.statusText,
      headers: {
        "Content-Type": contentType || "application/octet-stream",
        "Cache-Control": res.headers.get("cache-control") || "no-store",
      },
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Proxy request failed";
    const code = err && typeof (err as { code?: string }).code === "string" ? (err as { code: string }).code : "";
    return NextResponse.json(
      { error: message, ...(code === "ECONNREFUSED" && { details: "Backend unreachable; check BACKEND_URL and backend pod." }) },
      { status: 502 }
    );
  }
}

export function GET(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function POST(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function PUT(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function PATCH(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function DELETE(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function HEAD(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
export function OPTIONS(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return context.params.then((p) => proxy(request, p.path));
}
