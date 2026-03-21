import { api } from "./client";

export interface Product {
  id: string;
  workspace_id: string;
  sku?: string;
  name: string;
  category: string;
  default_fields?: Record<string, string | null>;
  print_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProductsListResponse {
  total: number;
  products: Product[];
}

export const productsApi = {
  list: (params?: { q?: string; category?: string; page?: number; per_page?: number }) =>
    api.get<ProductsListResponse>("/v1/products?" + new URLSearchParams(
      Object.fromEntries(Object.entries(params ?? {}).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)]))
    )),

  get: (id: string) =>
    api.get<Product>(`/v1/products/${id}`),

  create: (data: { sku?: string; name: string; category?: string; default_fields?: Record<string, string | null> }) =>
    api.post<Product>("/v1/products", data),

  update: (id: string, data: Partial<{ name: string; category: string; default_fields: Record<string, string | null> }>) =>
    api.patch<Product>(`/v1/products/${id}`, data),
};
