import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL =
  process.env.BACKEND_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080";

/** 视频上传专用代理：无超时，避免 rewrites 默认约 30 秒导致 socket hang up / 500 */
export async function POST(request: NextRequest) {
  if (!request.body) {
    return NextResponse.json({ error: "Request body required" }, { status: 400 });
  }
  const url = `${BACKEND_URL.replace(/\/$/, "")}/api/content/videos/upload`;
  const headers = new Headers();
  const auth = request.headers.get("authorization");
  if (auth) headers.set("Authorization", auth);
  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);

  const fetchOptions: RequestInit & { duplex?: "half" } = {
    method: "POST",
    body: request.body,
    headers,
    duplex: "half",
  };

  try {
    const res = await fetch(url, fetchOptions);
    const data = await res.json().catch(() => ({}));
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Proxy request failed";
    return NextResponse.json({ error: message }, { status: 502 });
  }
}
