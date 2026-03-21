"use client";

import { useState } from "react";
import { Loader2, UserPlus, Trash2 } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useWorkspaceProfile,
  useMembers,
  useUpdateProfile,
  useInviteMember,
  useRevokeMember,
} from "@/lib/hooks/useWorkspace";
import { useAuthStore } from "@/lib/stores/authStore";
import { inviteSchema, profileSchema, type InviteInput, type ProfileInput } from "@/lib/schemas/workspace";
import { hasRole } from "@/lib/utils/roleGuard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/utils/formatters";

function InviteDialog({ onClose }: { onClose: () => void }) {
  const inviteMut = useInviteMember();
  const { register, handleSubmit, setValue, watch, formState: { errors } } = useForm<InviteInput>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { email: "", role: "viewer" },
  });

  async function onSubmit(data: InviteInput) {
    await inviteMut.mutateAsync({ email: data.email, role: data.role });
    onClose();
  }

  return (
    <DialogContent className="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>Invită membru</DialogTitle>
      </DialogHeader>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="inv-email">Email</Label>
          <Input id="inv-email" type="email" {...register("email")} />
          {errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
        </div>
        <div className="space-y-1.5">
          <Label>Rol</Label>
          <Select value={watch("role")} onValueChange={(v) => setValue("role", v as InviteInput["role"])}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="viewer">Vizualizator</SelectItem>
              <SelectItem value="operator">Operator</SelectItem>
              <SelectItem value="admin">Administrator</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>Anulează</Button>
          <Button type="submit" disabled={inviteMut.isPending}>
            {inviteMut.isPending && <Loader2 className="animate-spin" />}
            Trimite invitație
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}

export default function WorkspacePage() {
  const role = useAuthStore((s) => s.user?.role);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [editMode, setEditMode] = useState(false);
  const [revokeId, setRevokeId] = useState<string | null>(null);

  const { data: profile, isPending: profilePending } = useWorkspaceProfile();
  const { data: membersData, isPending: membersPending } = useMembers();
  const updateProfile = useUpdateProfile();
  const revokeMut = useRevokeMember();

  const { register: regProfile, handleSubmit: handleProfileSubmit, reset: resetProfile, formState: { errors: profileErrors } } = useForm<ProfileInput>({
    resolver: zodResolver(profileSchema),
  });

  async function onSaveProfile(data: ProfileInput) {
    await updateProfile.mutateAsync({ name: data.name, cui: data.cui });
    setEditMode(false);
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Workspace</h1>

      {/* Profile */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Profil companie</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {profilePending ? (
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          ) : profile ? (
            editMode ? (
              <form onSubmit={handleProfileSubmit(onSaveProfile)} className="space-y-3">
                <div className="space-y-1.5">
                  <Label>Denumire</Label>
                  <Input {...regProfile("name")} defaultValue={profile?.name} />
                  {profileErrors.name && <p className="text-xs text-destructive">{profileErrors.name.message}</p>}
                </div>
                <div className="space-y-1.5">
                  <Label>CUI</Label>
                  <Input {...regProfile("cui")} defaultValue={profile?.cui} placeholder="RO12345678" />
                </div>
                <div className="flex gap-2">
                  <Button type="submit" size="sm" disabled={updateProfile.isPending}>Salvează</Button>
                  <Button type="button" variant="outline" size="sm" onClick={() => { setEditMode(false); resetProfile(); }}>Anulează</Button>
                </div>
              </form>
            ) : (
              <div className="space-y-2 text-sm">
                <div className="flex items-center gap-4">
                  <span className="w-28 text-muted-foreground">Denumire</span>
                  <span className="font-medium">{profile.name}</span>
                </div>
                {profile.cui && (
                  <div className="flex items-center gap-4">
                    <span className="w-28 text-muted-foreground">CUI</span>
                    <span>{profile.cui}</span>
                  </div>
                )}
                <div className="flex items-center gap-4">
                  <span className="w-28 text-muted-foreground">Plan</span>
                  <span className="capitalize">{profile.plan}</span>
                </div>
                <div className="flex items-center gap-4">
                  <span className="w-28 text-muted-foreground">Cotă etichete</span>
                  <span>{profile.labels_used} / {profile.label_quota}</span>
                </div>
                {hasRole(role, "admin") && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onClick={() => setEditMode(true)}
                  >
                    Editează
                  </Button>
                )}
              </div>
            )
          ) : null}
        </CardContent>
      </Card>

      {/* Members */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Membri</CardTitle>
          {hasRole(role, "admin") && (
            <Button size="sm" onClick={() => setInviteOpen(true)}>
              <UserPlus className="h-4 w-4" />
              Invită
            </Button>
          )}
        </CardHeader>
        <CardContent className="p-0">
          {membersPending ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : !membersData?.members?.length ? (
            <p className="py-8 text-center text-sm text-muted-foreground">Niciun membru.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">Email</th>
                  <th className="px-4 py-3 text-left font-medium">Rol</th>
                  <th className="px-4 py-3 text-left font-medium">Alăturat</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {membersData.members.map((m) => (
                  <tr key={m.id} className="border-b last:border-0">
                    <td className="px-4 py-3">{m.email}</td>
                    <td className="px-4 py-3 capitalize">{m.role}</td>
                    <td className="px-4 py-3 text-muted-foreground">{formatDate(m.joined_at)}</td>
                    <td className="px-4 py-3 text-right">
                      {hasRole(role, "admin") && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-destructive hover:text-destructive"
                          onClick={() => setRevokeId(m.id)}
                          disabled={revokeMut.isPending}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      <Dialog open={inviteOpen} onOpenChange={setInviteOpen}>
        <InviteDialog onClose={() => setInviteOpen(false)} />
      </Dialog>

      <AlertDialog open={!!revokeId} onOpenChange={(o) => !o && setRevokeId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoci accesul membrului?</AlertDialogTitle>
            <AlertDialogDescription>
              Membrul va pierde imediat accesul la workspace. Poți să-l reinviți oricând.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Anulează</AlertDialogCancel>
            <AlertDialogAction onClick={() => { revokeMut.mutate(revokeId!); setRevokeId(null); }}>
              Revocă accesul
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
