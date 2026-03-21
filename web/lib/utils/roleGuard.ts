import type { Role } from "@/lib/stores/authStore";

const LEVELS: Record<Role, number> = {
  viewer: 1,
  operator: 2,
  admin: 3,
};

/** Returns true if userRole meets or exceeds the minRole requirement */
export function hasRole(userRole: Role | undefined | null, minRole: Role): boolean {
  if (!userRole) return false;
  return LEVELS[userRole] >= LEVELS[minRole];
}
