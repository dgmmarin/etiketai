import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function POST(req: NextRequest) {
  const rt = req.cookies.get("rt")?.value;
  if (!rt) {
    return NextResponse.json({ error: "no refresh token" }, { status: 401 });
  }

  const upstream = await fetch(`${API_URL}/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: rt }),
  });

  if (!upstream.ok) {
    const res = NextResponse.json({ error: "invalid refresh token" }, { status: 401 });
    res.cookies.set("rt", "", { httpOnly: true, maxAge: 0, path: "/" });
    return res;
  }

  const data = await upstream.json();
  return NextResponse.json(data);
}