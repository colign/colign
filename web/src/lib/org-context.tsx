"use client";

import { createContext, useContext, useEffect, useState, useCallback } from "react";
import { orgClient } from "./organization";
import { saveTokens, getAccessToken } from "./auth";
import { showError } from "@/lib/toast";

interface Org {
  id: bigint;
  name: string;
  slug: string;
}

interface OrgContextValue {
  currentOrg: Org | null;
  orgs: Org[];
  loading: boolean;
  switchOrg: (orgId: bigint) => Promise<void>;
  refresh: () => Promise<void>;
}

const OrgContext = createContext<OrgContextValue>({
  currentOrg: null,
  orgs: [],
  loading: true,
  switchOrg: async () => {},
  refresh: async () => {},
});

export function OrgProvider({ children }: { children: React.ReactNode }) {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [currentOrgId, setCurrentOrgId] = useState<bigint | null>(null);
  const [loading, setLoading] = useState(true);

  const loadOrgs = useCallback(async () => {
    const token = getAccessToken();
    if (!token) {
      setLoading(false);
      return;
    }

    try {
      const res = await orgClient.listOrganizations({});
      const orgList = res.organizations.map((o) => ({
        id: o.id,
        name: o.name,
        slug: o.slug,
      }));
      setOrgs(orgList);
      setCurrentOrgId(res.currentOrgId);
    } catch (err) {
      showError("Failed to load organization", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadOrgs();
  }, [loadOrgs]);

  const switchOrg = useCallback(async (orgId: bigint) => {
    try {
      const res = await orgClient.switchOrganization({ organizationId: orgId });
      saveTokens(res.accessToken, res.refreshToken);
      setCurrentOrgId(orgId);
      // Always navigate to projects list since all pages show org-specific data
      window.location.href = "/projects";
    } catch (err) {
      showError("Failed to load organization", err);
    }
  }, []);

  const currentOrg = orgs.find((o) => o.id === currentOrgId) ?? null;

  return (
    <OrgContext.Provider value={{ currentOrg, orgs, loading, switchOrg, refresh: loadOrgs }}>
      {children}
    </OrgContext.Provider>
  );
}

export function useOrg() {
  return useContext(OrgContext);
}
