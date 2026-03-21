import { z } from "zod";

const CATEGORIES = ["food", "cosmetic", "electronics", "toy", "other"] as const;

export const productSchema = z.object({
  name: z.string().min(1, "Denumirea este obligatorie"),
  sku: z.string().optional(),
  category: z.enum(CATEGORIES).default("other"),
});

export type ProductInput = z.infer<typeof productSchema>;
