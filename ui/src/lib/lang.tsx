import { createContext, useContext, useMemo, useState, type ReactNode } from "react";

export type Lang = "en" | "my";

const LANG_STORAGE_KEY = "golangci_lang";

function getStoredLang(): Lang {
  const stored = localStorage.getItem(LANG_STORAGE_KEY);
  return stored === "en" || stored === "my" ? stored : "my";
}

const LangContext = createContext<{ lang: Lang; setLang: (l: Lang) => void } | null>(null);

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(getStoredLang);

  function setLang(l: Lang) {
    localStorage.setItem(LANG_STORAGE_KEY, l);
    setLangState(l);
  }

  const value = useMemo(() => ({ lang, setLang }), [lang]);
  return <LangContext.Provider value={value}>{children}</LangContext.Provider>;
}

export function useLang() {
  const ctx = useContext(LangContext);
  if (!ctx) throw new Error("useLang must be used within a LangProvider");
  return ctx;
}

// pick returns the Burmese value when lang is "my" and it's non-empty,
// falling back to the English value otherwise (older records predating
// this field, or a generation step that left it blank).
export function pick(lang: Lang, en: string | undefined, my: string | undefined): string {
  if (lang === "my" && my) return my;
  return en ?? "";
}
