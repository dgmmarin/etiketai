import { z } from "zod";

export const profileSchema = z.object({
  name: z.string().min(2, "Minim 2 caractere"),
  cui: z.string().optional(),
});

export const inviteSchema = z.object({
  email: z.string().email("Email invalid"),
  role: z.enum(["viewer", "operator", "admin"]).default("viewer"),
});

export type ProfileInput = z.infer<typeof profileSchema>;
export type InviteInput = z.infer<typeof inviteSchema>;
