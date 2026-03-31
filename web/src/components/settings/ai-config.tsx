"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { aiConfigClient } from "@/lib/aiconfig";
import { useI18n } from "@/lib/i18n";
import { showError, showSuccess } from "@/lib/toast";

type ModelOption = {
  value: string;
  label: string;
};

const MODEL_OPTIONS_BY_PROVIDER: Record<string, ModelOption[]> = {
  openai: [
    { value: "gpt-5.4", label: "GPT-5.4" },
    { value: "gpt-5.4-mini", label: "GPT-5.4 mini" },
    { value: "gpt-5.4-nano", label: "GPT-5.4 nano" },
    { value: "gpt-4o", label: "GPT-4o" },
    { value: "gpt-4o-mini", label: "GPT-4o mini" },
  ],
  anthropic: [
    { value: "claude-opus-4-6", label: "Claude Opus 4.6" },
    { value: "claude-sonnet-4-6", label: "Claude Sonnet 4.6" },
    { value: "claude-opus-4-1-20250805", label: "Claude Opus 4.1" },
    { value: "claude-opus-4-20250514", label: "Claude Opus 4" },
    { value: "claude-sonnet-4-20250514", label: "Claude Sonnet 4" },
    { value: "claude-3-7-sonnet-20250219", label: "Claude Sonnet 3.7" },
    { value: "claude-haiku-4-5-20251001", label: "Claude Haiku 4.5" },
    { value: "claude-3-5-haiku-20241022", label: "Claude Haiku 3.5" },
  ],
};

interface AIConfigSettingsProps {
  projectId: bigint;
}

export function AIConfigSettings({ projectId }: AIConfigSettingsProps) {
  const { t } = useI18n();

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [saved, setSaved] = useState(false);

  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [apiKeyMasked, setApiKeyMasked] = useState("");
  const [includeProjectContext, setIncludeProjectContext] = useState(false);
  const [hasProjectConfig, setHasProjectConfig] = useState(false);
  const [orgConfig, setOrgConfig] = useState<{ provider: string; model: string } | null>(null);

  useEffect(() => {
    setLoading(true);

    const loadProject = aiConfigClient
      .getAIConfig({ projectId })
      .then((res) => {
        if (res.config) {
          setProvider(res.config.provider);
          setModel(res.config.model);
          setApiKeyMasked(res.config.apiKeyMasked);
          setIncludeProjectContext(res.config.includeProjectContext);
          setHasProjectConfig(true);
        }
      })
      .catch((err: unknown) => {
        showError(t("toast.loadFailed"), err);
      });

    const loadOrg = aiConfigClient
      .getOrgAIConfig({})
      .then((res) => {
        if (res.config && res.config.provider) {
          setOrgConfig({ provider: res.config.provider, model: res.config.model });
        }
      })
      .catch(() => {
        // Org config may not exist
      });

    Promise.all([loadProject, loadOrg]).finally(() => {
        setLoading(false);
      });
  }, [projectId, t]);

  function handleProviderChange(value: string | null) {
    if (!value) return;
    setProvider(value);
    setModel("");
  }

  const providerModels = provider ? (MODEL_OPTIONS_BY_PROVIDER[provider] ?? []) : [];
  const availableModels =
    model && providerModels.every((option) => option.value !== model)
      ? [...providerModels, { value: model, label: `${model} (custom)` }]
      : providerModels;

  async function handleTestConnection() {
    if (!provider || !model) return;
    setTesting(true);
    try {
      const res = await aiConfigClient.testConnection({
        provider,
        model,
        apiKey,
      });
      if (res.success) {
        showSuccess(t("aiConfig.testSuccess"));
      } else {
        showError(t("aiConfig.testFailed"), res.error ? new Error(res.error) : undefined);
      }
    } catch (err: unknown) {
      showError(t("aiConfig.testFailed"), err);
    } finally {
      setTesting(false);
    }
  }

  async function handleSave() {
    if (!provider || !model) return;
    setSaving(true);
    try {
      const res = await aiConfigClient.saveAIConfig({
        projectId,
        provider,
        model,
        apiKey,
        includeProjectContext,
      });
      if (res.config) {
        setApiKeyMasked(res.config.apiKeyMasked);
        setApiKey("");
      }
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      showSuccess(t("toast.saveSuccess"));
    } catch (err: unknown) {
      showError(t("toast.saveFailed"), err);
    } finally {
      setSaving(false);
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
    <Card className="border-border/50">
      <CardHeader>
        <CardTitle>{t("aiConfig.title")}</CardTitle>
        <CardDescription>{t("aiConfig.notConfigured")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* Org config notice */}
        {orgConfig && !hasProjectConfig && (
          <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 px-4 py-3">
            <p className="text-sm text-blue-400">
              {t("aiConfig.orgConfigActive", { provider: orgConfig.provider, model: orgConfig.model })}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("aiConfig.orgConfigHint")}
            </p>
          </div>
        )}

        {/* Provider */}
        <div className="space-y-2">
          <Label>{t("aiConfig.provider")}</Label>
          <Select value={provider} onValueChange={handleProviderChange}>
            <SelectTrigger className="cursor-pointer">
              <SelectValue placeholder={t("aiConfig.provider")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="openai" className="cursor-pointer">
                OpenAI
              </SelectItem>
              <SelectItem value="anthropic" className="cursor-pointer">
                Anthropic
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Model */}
        <div className="space-y-2">
          <Label>{t("aiConfig.model")}</Label>
          <Select value={model} onValueChange={(v) => v && setModel(v)} disabled={!provider}>
            <SelectTrigger className="cursor-pointer">
              <SelectValue placeholder={t("aiConfig.model")} />
            </SelectTrigger>
            <SelectContent>
              {availableModels.map((option) => (
                <SelectItem key={option.value} value={option.value} className="cursor-pointer">
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* API Key */}
        <div className="space-y-2">
          <Label>{t("aiConfig.apiKey")}</Label>
          <Input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={apiKeyMasked || t("aiConfig.apiKeyPlaceholder")}
          />
        </div>

        {/* Include project context */}
        <div className="flex items-center gap-3">
          <Switch checked={includeProjectContext} onCheckedChange={setIncludeProjectContext} />
          <div>
            <p className="text-sm font-medium">{t("aiConfig.includeContext")}</p>
            <p className="text-xs text-muted-foreground">{t("aiConfig.includeContextHelp")}</p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3 pt-2">
          <Button
            variant="outline"
            onClick={handleTestConnection}
            disabled={testing || !provider || !model}
            className="cursor-pointer"
          >
            {testing ? t("common.loading") : t("aiConfig.testConnection")}
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !provider || !model}
            className="cursor-pointer"
          >
            {saving ? t("common.saving") : t("aiConfig.save")}
          </Button>
          {saved && <span className="text-sm text-emerald-400">{t("common.saved")}</span>}
        </div>
      </CardContent>
    </Card>
  );
}
