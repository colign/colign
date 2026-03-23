"use client";

import { useState, useEffect, useCallback } from "react";
import { orgClient } from "@/lib/organization";
import { useOrg } from "@/lib/org-context";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { UserPlus, Trash2, Shield, User, Crown, X, Mail, Globe, Copy, Check } from "lucide-react";

type Member = {
  id: bigint;
  userId: bigint;
  userName: string;
  userEmail: string;
  role: string;
};

type Invitation = {
  id: bigint;
  email: string;
  role: string;
  status: string;
  token: string;
};

const roleIcons: Record<string, typeof Crown> = {
  owner: Crown,
  admin: Shield,
  member: User,
};

const roleLabels: Record<string, string> = {
  owner: "Owner",
  admin: "Admin",
  member: "Member",
};

export function OrgMembers() {
  const { currentOrg } = useOrg();
  const [members, setMembers] = useState<Member[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [loading, setLoading] = useState(true);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviting, setInviting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [copied, setCopied] = useState(false);

  // Domain settings
  const [domains, setDomains] = useState<string[]>([]);
  const [newDomain, setNewDomain] = useState("");
  const [savingDomains, setSavingDomains] = useState(false);

  const fetchMembers = useCallback(async () => {
    try {
      const res = await orgClient.listMembers({});
      setMembers(
        res.members.map((m) => ({
          id: m.id,
          userId: m.userId,
          userName: m.userName,
          userEmail: m.userEmail,
          role: m.role,
        })),
      );
    } catch (err) {
      console.error("Failed to load members:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchInvitations = useCallback(async () => {
    try {
      const res = await orgClient.listInvitations({});
      setInvitations(
        res.invitations.map((inv) => ({
          id: inv.id,
          email: inv.email,
          role: inv.role,
          status: inv.status,
          token: inv.token,
        })),
      );
    } catch (err) {
      console.error("Failed to load invitations:", err);
    }
  }, []);

  const fetchOrgDetails = useCallback(async () => {
    try {
      const res = await orgClient.listOrganizations({});
      const org = res.organizations.find((o) => o.id === res.currentOrgId);
      if (org) {
        setDomains([...org.allowedDomains]);
      }
    } catch (err) {
      console.error("Failed to load org details:", err);
    }
  }, []);

  useEffect(() => {
    fetchMembers();
    fetchInvitations();
    fetchOrgDetails();
  }, [fetchMembers, fetchInvitations, fetchOrgDetails]);

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    setInviting(true);
    setError("");
    setSuccess("");
    try {
      const res = await orgClient.inviteOrgMember({
        email: inviteEmail.trim(),
        role: inviteRole,
      });
      const link = `${window.location.origin}/invite/${res.invitation?.token}`;
      setInviteLink(link);
      setCopied(false);
      setSuccess("Invitation sent!");
      setInviteEmail("");
      fetchInvitations();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to invite");
    } finally {
      setInviting(false);
    }
  }

  async function handleRevokeInvitation(invitationId: bigint) {
    try {
      await orgClient.revokeInvitation({ invitationId });
      fetchInvitations();
    } catch (err) {
      console.error("Failed to revoke invitation:", err);
    }
  }

  async function handleRemove(userId: bigint, name: string) {
    if (!confirm(`Remove ${name} from the organization?`)) return;
    try {
      await orgClient.removeOrgMember({ userId });
      fetchMembers();
    } catch (err) {
      console.error("Failed to remove member:", err);
    }
  }

  async function handleRoleChange(userId: bigint, role: string) {
    try {
      await orgClient.updateOrgMemberRole({ userId, role });
      fetchMembers();
    } catch (err) {
      console.error("Failed to update role:", err);
    }
  }

  function handleAddDomain() {
    const domain = newDomain.trim().toLowerCase();
    if (!domain || domains.includes(domain)) return;
    setDomains([...domains, domain]);
    setNewDomain("");
  }

  function handleRemoveDomain(domain: string) {
    setDomains(domains.filter((d) => d !== domain));
  }

  async function handleSaveDomains() {
    setSavingDomains(true);
    try {
      await orgClient.setAllowedDomains({ domains });
      setSuccess("Allowed domains updated");
      setTimeout(() => setSuccess(""), 3000);
    } catch (err) {
      console.error("Failed to save domains:", err);
    } finally {
      setSavingDomains(false);
    }
  }

  if (loading) {
    return (
      <Card className="border-border/50">
        <CardContent className="flex items-center justify-center py-12">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Domain-based auto-join */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="size-5" />
            Allowed Email Domains
          </CardTitle>
          <CardDescription>
            Users with these email domains will automatically join your organization when they sign
            up.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="example.com"
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), handleAddDomain())}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              onClick={handleAddDomain}
              disabled={!newDomain.trim()}
              className="cursor-pointer"
            >
              Add
            </Button>
          </div>
          {domains.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {domains.map((domain) => (
                <span
                  key={domain}
                  className="inline-flex items-center gap-1.5 rounded-md bg-accent px-2.5 py-1 text-sm"
                >
                  @{domain}
                  <button
                    onClick={() => handleRemoveDomain(domain)}
                    className="cursor-pointer rounded-sm text-muted-foreground hover:text-foreground"
                  >
                    <X className="size-3" />
                  </button>
                </span>
              ))}
            </div>
          )}
          <Button
            onClick={handleSaveDomains}
            disabled={savingDomains}
            size="sm"
            className="cursor-pointer"
          >
            {savingDomains ? "Saving..." : "Save Domains"}
          </Button>
        </CardContent>
      </Card>

      {/* Members & Invitations */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>Organization Members</CardTitle>
          <CardDescription>
            Manage members of {currentOrg?.name ?? "your organization"}.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Invite form */}
          <form onSubmit={handleInvite} className="flex gap-2">
            <Input
              type="email"
              placeholder="Email address"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              className="flex-1"
            />
            <select
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value)}
              className="rounded-md border border-border bg-background px-3 py-2 text-sm"
            >
              <option value="member">Member</option>
              <option value="admin">Admin</option>
            </select>
            <Button
              type="submit"
              disabled={inviting || !inviteEmail.trim()}
              className="cursor-pointer"
            >
              <UserPlus className="mr-1.5 size-4" />
              {inviting ? "Inviting..." : "Invite"}
            </Button>
          </form>

          {error && <p className="text-sm text-destructive">{error}</p>}
          {success && inviteLink && (
            <div className="flex items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2">
              <p className="flex-1 truncate text-sm text-emerald-400">{inviteLink}</p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 shrink-0 cursor-pointer gap-1.5 px-2 text-xs"
                onClick={async () => {
                  await navigator.clipboard.writeText(inviteLink);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 2000);
                }}
              >
                {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
                {copied ? "Copied" : "Copy"}
              </Button>
              <button
                onClick={() => { setSuccess(""); setInviteLink(""); }}
                className="cursor-pointer rounded-sm p-0.5 text-muted-foreground hover:text-foreground"
              >
                <X className="size-3.5" />
              </button>
            </div>
          )}

          {/* Pending Invitations */}
          {invitations.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Pending Invitations</p>
              <div className="divide-y divide-border/50 rounded-lg border border-dashed border-border/50">
                {invitations.map((inv) => (
                  <div key={String(inv.id)} className="flex items-center justify-between px-4 py-3">
                    <div className="flex items-center gap-3">
                      <Mail className="size-4 text-muted-foreground" />
                      <div>
                        <p className="text-sm">{inv.email}</p>
                        <p className="text-xs text-muted-foreground">
                          {roleLabels[inv.role] ?? inv.role} &middot; Pending
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRevokeInvitation(inv.id)}
                      className="cursor-pointer rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                    >
                      <X className="size-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Members list */}
          <div className="divide-y divide-border/50 rounded-lg border border-border/50">
            {members.map((member) => {
              const RoleIcon = roleIcons[member.role] ?? User;
              return (
                <div
                  key={String(member.id)}
                  className="flex items-center justify-between px-4 py-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex size-8 items-center justify-center rounded-full bg-accent text-xs font-medium">
                      {member.userName?.[0]?.toUpperCase() ??
                        member.userEmail?.[0]?.toUpperCase() ??
                        "?"}
                    </div>
                    <div>
                      <p className="text-sm font-medium">{member.userName || "\u2014"}</p>
                      <p className="text-xs text-muted-foreground">{member.userEmail}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {member.role === "owner" ? (
                      <span className="flex items-center gap-1 rounded-md bg-amber-500/10 px-2 py-1 text-xs text-amber-400">
                        <RoleIcon className="size-3" />
                        {roleLabels[member.role]}
                      </span>
                    ) : (
                      <>
                        <select
                          value={member.role}
                          onChange={(e) => handleRoleChange(member.userId, e.target.value)}
                          className="cursor-pointer rounded-md border border-border/50 bg-background px-2 py-1 text-xs"
                        >
                          <option value="member">Member</option>
                          <option value="admin">Admin</option>
                        </select>
                        <button
                          onClick={() =>
                            handleRemove(member.userId, member.userName || member.userEmail)
                          }
                          className="cursor-pointer rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                        >
                          <Trash2 className="size-3.5" />
                        </button>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
            {members.length === 0 && (
              <div className="px-4 py-8 text-center text-sm text-muted-foreground">
                No members yet. Invite someone to get started.
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
