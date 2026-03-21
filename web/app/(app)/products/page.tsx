"use client";

import { useState } from "react";
import { Plus, Pencil, Loader2 } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useProducts, useCreateProduct, useUpdateProduct } from "@/lib/hooks/useProducts";
import { productSchema, type ProductInput } from "@/lib/schemas/product";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/utils/formatters";
import type { Product } from "@/lib/api/products";

const CATEGORIES = ["food", "cosmetic", "electronics", "toy", "other"];

function ProductDialog({
  product,
  onClose,
}: {
  product?: Product;
  onClose: () => void;
}) {
  const createMut = useCreateProduct();
  const updateMut = useUpdateProduct();
  const isPending = createMut.isPending || updateMut.isPending;

  const { register, handleSubmit, setValue, watch, formState: { errors } } = useForm<ProductInput>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      name: product?.name ?? "",
      sku: product?.sku ?? "",
      category: (product?.category as ProductInput["category"]) ?? "other",
    },
  });

  async function onSubmit(data: ProductInput) {
    if (product) {
      await updateMut.mutateAsync({ id: product.id, data: { name: data.name, category: data.category } });
    } else {
      await createMut.mutateAsync({ name: data.name, sku: data.sku || undefined, category: data.category });
    }
    onClose();
  }

  return (
    <DialogContent className="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{product ? "Editează produs" : "Produs nou"}</DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="name">Denumire *</Label>
          <Input id="name" {...register("name")} />
          {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="sku">SKU</Label>
          <Input id="sku" {...register("sku")} placeholder="Ex: LAP-001" />
        </div>
        <div className="space-y-1.5">
          <Label>Categorie</Label>
          <Select value={watch("category")} onValueChange={(v) => setValue("category", v as ProductInput["category"])}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CATEGORIES.map((c) => (
                <SelectItem key={c} value={c}>{c}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>Anulează</Button>
          <Button type="submit" disabled={isPending}>
            {isPending && <Loader2 className="animate-spin" />}
            {product ? "Salvează" : "Creează"}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}

export default function ProductsPage() {
  const [query, setQuery] = useState("");
  const [dialogProduct, setDialogProduct] = useState<Product | null | "new">(null);

  const { data, isPending } = useProducts({ q: query || undefined });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Produse</h1>
        <Button onClick={() => setDialogProduct("new")}>
          <Plus /> Produs nou
        </Button>
      </div>

      <Input
        placeholder="Caută după nume sau SKU..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        className="max-w-sm"
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{data ? `${data.total} produse` : "Produse"}</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isPending ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : !data?.products?.length ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              Niciun produs în bibliotecă.
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">Denumire</th>
                  <th className="px-4 py-3 text-left font-medium">SKU</th>
                  <th className="px-4 py-3 text-left font-medium">Categorie</th>
                  <th className="px-4 py-3 text-left font-medium">Tipăriri</th>
                  <th className="px-4 py-3 text-left font-medium">Adăugat</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {data.products.map((p) => (
                  <tr key={p.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-medium">{p.name}</td>
                    <td className="px-4 py-3 text-muted-foreground">{p.sku ?? "—"}</td>
                    <td className="px-4 py-3 capitalize">{p.category}</td>
                    <td className="px-4 py-3">{p.print_count}</td>
                    <td className="px-4 py-3 text-muted-foreground">{formatDate(p.created_at)}</td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => setDialogProduct(p)}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogProduct !== null} onOpenChange={(o) => !o && setDialogProduct(null)}>
        {dialogProduct !== null && (
          <ProductDialog
            product={dialogProduct !== "new" ? dialogProduct : undefined}
            onClose={() => setDialogProduct(null)}
          />
        )}
      </Dialog>
    </div>
  );
}
