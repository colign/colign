"use client";

import { startTransition, useCallback } from "react";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { defaultLocale, isLocale, localeCookieName, type Locale } from "./config";

type TranslationParams = Record<string, string | number | Date>;

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, params?: TranslationParams) => string;
}

function writeLocaleCookie(locale: Locale) {
  document.cookie = `${localeCookieName}=${locale}; path=/; max-age=31536000; samesite=lax`;
}

export function useI18n(): I18nContextValue {
  const translations = useTranslations();
  const activeLocale = useLocale();
  const router = useRouter();

  const locale = isLocale(activeLocale) ? activeLocale : defaultLocale;

  const setLocale = useCallback(
    (nextLocale: Locale) => {
      if (nextLocale === locale) return;
      writeLocaleCookie(nextLocale);
      startTransition(() => {
        router.refresh();
      });
    },
    [locale, router],
  );

  const t = useCallback(
    (key: string, params?: TranslationParams) => translations(key, params),
    [translations],
  );

  return {
    locale,
    setLocale,
    t,
  };
}

export type { Locale };
