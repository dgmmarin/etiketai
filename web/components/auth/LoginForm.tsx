"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import Script from "next/script";
import { authApi, storeRefreshCookie, toAuthUser } from "@/lib/api/auth";
import { loginSchema, type LoginInput } from "@/lib/schemas/auth";
import { useAuthStore } from "@/lib/stores/authStore";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (cfg: object) => void;
          renderButton: (el: HTMLElement, cfg: object) => void;
        };
      };
    };
  }
}

const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { setSession } = useAuthStore();
  const googleBtnRef = useRef<HTMLDivElement>(null);
  const [googleError, setGoogleError] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginInput>({ resolver: zodResolver(loginSchema) });

  const from = searchParams.get("from") ?? "/";

  const handleGoogleCredential = useCallback(
    async (response: { credential: string }) => {
      setGoogleError("");
      try {
        const res = await authApi.oauthGoogle(response.credential);
        await storeRefreshCookie(res.refresh_token);
        setSession(res.access_token, toAuthUser(res.user));
        router.push(from);
      } catch (err: unknown) {
        setGoogleError(err instanceof Error ? err.message : "Autentificare Google eșuată");
      }
    },
    [router, setSession, from],
  );

  const initGoogle = useCallback(() => {
    if (!GOOGLE_CLIENT_ID || !window.google || !googleBtnRef.current) return;
    window.google.accounts.id.initialize({
      client_id: GOOGLE_CLIENT_ID,
      callback: handleGoogleCredential,
    });
    window.google.accounts.id.renderButton(googleBtnRef.current, {
      type: "standard",
      theme: "outline",
      size: "large",
      width: googleBtnRef.current.offsetWidth,
      text: "signin_with",
      locale: "ro",
    });
  }, [handleGoogleCredential]);

  // Re-init if GSI already loaded (e.g. navigating back)
  useEffect(() => {
    if (window.google) initGoogle();
  }, [initGoogle]);

  async function onSubmit(data: LoginInput) {
    try {
      const res = await authApi.login(data.email, data.password);
      await storeRefreshCookie(res.refresh_token);
      setSession(res.access_token, toAuthUser(res.user));
      router.push(from);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Eroare la autentificare";
      setError("root", { message: msg });
    }
  }

  return (
    <>
      {GOOGLE_CLIENT_ID && (
        <Script
          src="https://accounts.google.com/gsi/client"
          strategy="lazyOnload"
          onLoad={initGoogle}
        />
      )}
      <Card>
        <CardHeader>
          <CardTitle>Autentificare</CardTitle>
          <CardDescription>Intră în contul tău EtiketAI</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {GOOGLE_CLIENT_ID && (
            <>
              <div ref={googleBtnRef} className="w-full" />
              {googleError && (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {googleError}
                </p>
              )}
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">sau</span>
                </div>
              </div>
            </>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" autoComplete="email" {...register("email")} />
              {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="password">Parolă</Label>
              <Input id="password" type="password" autoComplete="current-password" {...register("password")} />
              {errors.password && <p className="text-xs text-destructive">{errors.password.message}</p>}
            </div>

            {errors.root && (
              <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {errors.root.message}
              </p>
            )}

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? "Se autentifică..." : "Intră în cont"}
            </Button>

            <p className="text-center text-sm text-muted-foreground">
              Nu ai cont?{" "}
              <Link href="/register" className="text-primary underline-offset-4 hover:underline">
                Înregistrare
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </>
  );
}
