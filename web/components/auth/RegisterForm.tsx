"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { authApi } from "@/lib/api/auth";
import { registerSchema, type RegisterInput } from "@/lib/schemas/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function RegisterForm() {
  const router = useRouter();
  const [success, setSuccess] = useState(false);

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<RegisterInput>({ resolver: zodResolver(registerSchema) });

  async function onSubmit(data: RegisterInput) {
    try {
      await authApi.register(data.email, data.password, data.workspace_name, data.cui);
      setSuccess(true);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Eroare la înregistrare";
      setError("root", { message: msg });
    }
  }

  if (success) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Verifică emailul</CardTitle>
          <CardDescription>
            Ți-am trimis un email de confirmare. Verifică-ți inbox-ul pentru a activa contul.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline" className="w-full" onClick={() => router.push("/login")}>
            Înapoi la autentificare
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Creare cont</CardTitle>
        <CardDescription>Începe cu EtiketAI — primele 14 zile gratuite</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" autoComplete="email" {...register("email")} />
            {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="password">Parolă</Label>
            <Input id="password" type="password" autoComplete="new-password" {...register("password")} />
            {errors.password && <p className="text-xs text-destructive">{errors.password.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="workspace_name">Numele companiei</Label>
            <Input id="workspace_name" placeholder="Ex: Importuri SRL" {...register("workspace_name")} />
            {errors.workspace_name && <p className="text-xs text-destructive">{errors.workspace_name.message}</p>}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="cui">CUI (opțional)</Label>
            <Input id="cui" placeholder="RO12345678" {...register("cui")} />
          </div>

          {errors.root && (
            <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {errors.root.message}
            </p>
          )}

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? "Se creează contul..." : "Creează cont"}
          </Button>

          <p className="text-center text-sm text-muted-foreground">
            Ai deja cont?{" "}
            <Link href="/login" className="text-primary underline-offset-4 hover:underline">
              Autentificare
            </Link>
          </p>
        </form>
      </CardContent>
    </Card>
  );
}
