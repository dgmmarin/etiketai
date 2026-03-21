"use client";

import { useState } from "react";
import { Loader2, CheckCircle } from "lucide-react";
import { useSubscription } from "@/lib/hooks/useWorkspace";
import { billingApi } from "@/lib/api/billing";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { formatDateShort } from "@/lib/utils/formatters";

const PLANS = [
  {
    id: "starter",
    name: "Starter",
    price: "49 lei/lună",
    quota: "100 etichete/lună",
    features: ["Procesare AI", "Export CSV", "Suport email"],
  },
  {
    id: "business",
    name: "Business",
    price: "149 lei/lună",
    quota: "500 etichete/lună",
    features: ["Tot din Starter", "Tipărire PDF/ZPL", "Membrii echipă (5)", "API access"],
    popular: true,
  },
  {
    id: "enterprise",
    name: "Enterprise",
    price: "399 lei/lună",
    quota: "Nelimitat",
    features: ["Tot din Business", "Ollama on-premise", "GDPR complet", "SLA 99.9%", "Suport dedicat"],
  },
];

export default function BillingPage() {
  const { data: sub, isPending } = useSubscription();
  const [loading, setLoading] = useState<string | null>(null);

  async function handleCheckout(plan: string) {
    setLoading(plan);
    try {
      const { checkout_url } = await billingApi.createCheckout(plan);
      window.location.href = checkout_url;
    } catch {
      setLoading(null);
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Abonament</h1>

      {isPending ? (
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      ) : sub ? (
        <Card className="max-w-sm">
          <CardHeader>
            <CardTitle className="text-base">Plan curent</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Plan</span>
              <span className="font-medium capitalize">{sub.plan}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Status</span>
              <span className="capitalize">{sub.status}</span>
            </div>
            {sub.current_period_end && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Reînnoire</span>
                <span>{formatDateShort(sub.current_period_end)}</span>
              </div>
            )}
            {sub.cancel_at_period_end && (
              <p className="text-xs text-orange-600">Abonamentul se va anula la sfârșitul perioadei.</p>
            )}
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-3">
        {PLANS.map((plan) => (
          <Card key={plan.id} className={plan.popular ? "border-primary shadow-md" : ""}>
            <CardHeader>
              <CardTitle className="text-lg">{plan.name}</CardTitle>
              <CardDescription>{plan.price}</CardDescription>
              <p className="text-sm text-muted-foreground">{plan.quota}</p>
              {plan.popular && (
                <span className="inline-block rounded-full bg-primary px-2 py-0.5 text-xs font-medium text-primary-foreground w-fit">
                  Popular
                </span>
              )}
            </CardHeader>
            <CardContent>
              <ul className="space-y-1.5">
                {plan.features.map((f) => (
                  <li key={f} className="flex items-center gap-2 text-sm">
                    <CheckCircle className="h-4 w-4 text-green-500 shrink-0" />
                    {f}
                  </li>
                ))}
              </ul>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full"
                variant={sub?.plan === plan.id ? "secondary" : "default"}
                onClick={() => handleCheckout(plan.id)}
                disabled={loading === plan.id || sub?.plan === plan.id}
              >
                {loading === plan.id ? (
                  <Loader2 className="animate-spin" />
                ) : sub?.plan === plan.id ? (
                  "Plan activ"
                ) : (
                  "Selectează"
                )}
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>
    </div>
  );
}
