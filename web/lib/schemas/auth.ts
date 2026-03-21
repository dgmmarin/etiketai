import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().email("Email invalid"),
  password: z.string().min(1, "Parola este obligatorie"),
});

export const registerSchema = z.object({
  email: z.string().email("Email invalid"),
  password: z.string().min(8, "Minim 8 caractere"),
  workspace_name: z.string().min(2, "Minim 2 caractere"),
  cui: z.string().optional(),
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
