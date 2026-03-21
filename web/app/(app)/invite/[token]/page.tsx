"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { CheckCircle, XCircle, Loader2 } from "lucide-react";
import { workspaceApi } from "@/lib/api/workspace";
import { useAuthStore } from "@/lib/stores/authStore";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type State = "loading" | "success" | "error";

export default function InvitePage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [state, setState] = useState<State>("loading");
  const [result, setResult] = useState<{ workspace_id: string; role: string } | null>(null);
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    workspaceApi
      .acceptInvitation(token)
      .then((res) => {
        setResult(res);
        setState("success");
      })
      .catch((err: Error) => {
        setErrorMsg(err.message ?? "Invitația este invalidă sau a expirat.");
        setState("error");
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <Card className="w-full max-w-sm">
        {state === "loading" && (
          <>
            <CardHeader>
              <CardTitle>Se verifică invitația…</CardTitle>
            </CardHeader>
            <CardContent className="flex justify-center py-6">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </CardContent>
          </>
        )}

        {state === "success" && result && (
          <>
            <CardHeader>
              <div className="flex items-center gap-2 text-green-600">
                <CheckCircle className="h-6 w-6" />
                <CardTitle>Invitație acceptată!</CardTitle>
              </div>
              <CardDescription>
                Ai acum rol de <strong className="capitalize">{result.role}</strong> în workspace.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Ești autentificat ca <strong>{user?.email}</strong>. Poți accesa dashboard-ul acum.
              </p>
              <Button className="w-full" onClick={() => router.push("/")}>
                Mergi la dashboard
              </Button>
            </CardContent>
          </>
        )}

        {state === "error" && (
          <>
            <CardHeader>
              <div className="flex items-center gap-2 text-destructive">
                <XCircle className="h-6 w-6" />
                <CardTitle>Invitație invalidă</CardTitle>
              </div>
              <CardDescription>{errorMsg}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Invitația poate fi expirată (valabilă 48h) sau ai acceptat-o deja.
                Cere administratorului să trimită o invitație nouă.
              </p>
              <Button variant="outline" className="w-full" onClick={() => router.push("/")}>
                Înapoi la dashboard
              </Button>
            </CardContent>
          </>
        )}
      </Card>
    </div>
  );
}
