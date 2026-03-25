import en from "./locales/en.json";
import ko from "./locales/ko.json";

export const locales = ["en", "ko"] as const;
export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = "en";
export const localeCookieName = "colign_locale";

export const messagesByLocale = {
  en,
  ko,
} as const;

export function isLocale(value: string | null | undefined): value is Locale {
  return value === "en" || value === "ko";
}

export function getMessagesForLocale(locale: Locale) {
  return messagesByLocale[locale];
}
