"use client";

import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from "react";
import en from "./locales/en.json";
import ko from "./locales/ko.json";

const locales: Record<string, Record<string, Record<string, string>>> = { en, ko };

export type Locale = "en" | "ko";

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    const stored = localStorage.getItem("colign_locale") as Locale | null;
    if (stored && stored !== "en") {
      setLocaleState(stored);
    }
  }, []);

  const setLocale = useCallback((newLocale: Locale) => {
    setLocaleState(newLocale);
    localStorage.setItem("colign_locale", newLocale);
  }, []);

  const t = useCallback(
    (key: string): string => {
      // key format: "section.key" e.g. "common.save"
      const parts = key.split(".");
      if (parts.length !== 2) return key;

      const [section, field] = parts;
      const messages = locales[locale] ?? locales.en;
      return messages?.[section]?.[field] ?? locales.en?.[section]?.[field] ?? key;
    },
    [locale],
  );

  return <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
