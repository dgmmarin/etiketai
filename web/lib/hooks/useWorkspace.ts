import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { workspaceApi } from "@/lib/api/workspace";

export const workspaceKeys = {
  profile: ["workspace", "profile"] as const,
  subscription: ["workspace", "subscription"] as const,
  members: ["workspace", "members"] as const,
};

export function useWorkspaceProfile() {
  return useQuery({
    queryKey: workspaceKeys.profile,
    queryFn: workspaceApi.getProfile,
    staleTime: 5 * 60_000,
  });
}

export function useSubscription() {
  return useQuery({
    queryKey: workspaceKeys.subscription,
    queryFn: workspaceApi.getSubscription,
    staleTime: 5 * 60_000,
  });
}

export function useMembers() {
  return useQuery({
    queryKey: workspaceKeys.members,
    queryFn: workspaceApi.listMembers,
  });
}

export function useInviteMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ email, role }: { email: string; role: string }) =>
      workspaceApi.inviteMember(email, role),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceKeys.members }),
  });
}

export function useRevokeMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (memberId: string) => workspaceApi.revokeMember(memberId),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceKeys.members }),
  });
}

export function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof workspaceApi.updateProfile>[0]) =>
      workspaceApi.updateProfile(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceKeys.profile }),
  });
}
