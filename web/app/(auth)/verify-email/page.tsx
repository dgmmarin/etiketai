"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { CheckCircle, XCircle, Loader2 } from "lucide-react";
import Link from "next/link";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type State = "loading" | "success" | "error";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [state, setState] = useState<State>("loading");
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    if (!token) {
      setErrorMsg("Token lipsă sau invalid.");
      setState("error");
      return;
    }
    api
      .get<{ success: boolean }>(`/v1/auth/verify-email?token=${encodeURIComponent(token)}`)
      .then(() => setState("success"))
      .catch((err: Error) => {
        setErrorMsg(err.message ?? "Token invalid sau expirat.");
        setState("error");
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  return (
    <Card>
      {state === "loading" && (
        <>
          <CardHeader>
            <CardTitle>Se verifică emailul…</CardTitle>
          </CardHeader>
          <CardContent className="flex justify-center py-6">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </CardContent>
        </>
      )}

      {state === "success" && (
        <>
          <CardHeader>
            <div className="flex items-center gap-2 text-green-600">
              <CheckCircle className="h-6 w-6" />
              <CardTitle>Email verificat!</CardTitle>
            </div>
            <CardDescription>
              Adresa ta de email a fost confirmată. Te poți autentifica acum.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild className="w-full">
              <Link href="/login">Intră în cont</Link>
            </Button>
          </CardContent>
        </>
      )}

      {state === "error" && (
        <>
          <CardHeader>
            <div className="flex items-center gap-2 text-destructive">
              <XCircle className="h-6 w-6" />
              <CardTitle>Verificare eșuată</CardTitle>
            </div>
            <CardDescription>{errorMsg}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Link-ul poate fi expirat sau deja folosit. Contactează suportul dacă problema persistă.
            </p>
            <Button variant="outline" asChild className="w-full">
              <Link href="/login">Înapoi la autentificare</Link>
            </Button>
          </CardContent>
        </>
      )}
    </Card>
  );
}

export default function VerifyEmailPage() {
  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="w-full max-w-sm">
        <Suspense fallback={<Card><CardContent className="flex justify-center py-10"><Loader2 className="h-8 w-8 animate-spin text-muted-foreground" /></CardContent></Card>}>
          <VerifyEmailContent />
        </Suspense>
      </div>
    </div>
  );
}
