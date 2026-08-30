"use client";

import {
  BRANDING_COOKIE,
  DEFAULT_BRANDING,
  parseBrandingCookie,
  serializeBrandingCookie,
  type SystemBrandingConfig,
} from "./brandingConfig";
import { deleteCookie, readCookie, writeCookie } from "@/lib/prefs/cookies";

// "External store" (useSyncExternalStore) do branding white-label. Persiste
// num COOKIE, não em localStorage: o layout do servidor lê o mesmo cookie e
// já entrega a Topbar/Footer com o nome certo no 1º paint. O 3º argumento
// do useSyncExternalStore (server snapshot) do BrandingProvider é montado a
// partir desse mesmo cookie, então o snapshot de SSR e o do cliente batem —
// sem troca visível logo após a hidratação (o "flash do nome antigo").
//
// Migração: versões anteriores gravavam em localStorage
// ("nova_system_branding_v1"). Na 1ª leitura sem cookie, se houver um valor
// legado no localStorage, ele é promovido para cookie e o antigo é
// apagado — depois disso o SSR já nasce correto.

const LEGACY_STORAGE_KEY = "nova_system_branding_v1";

type Listener = () => void;
const listeners = new Set<Listener>();
let cached: SystemBrandingConfig | null = null;

function migrateLegacyStorage(): SystemBrandingConfig | null {
  try {
    const raw = window.localStorage.getItem(LEGACY_STORAGE_KEY);
    if (!raw) return null;
    const migrated = { ...DEFAULT_BRANDING, ...(JSON.parse(raw) as Partial<SystemBrandingConfig>) };
    writeCookie(BRANDING_COOKIE, serializeBrandingCookie(migrated));
    window.localStorage.removeItem(LEGACY_STORAGE_KEY);
    return migrated;
  } catch {
    return null;
  }
}

function read(): SystemBrandingConfig {
  if (cached) return cached;
  const fromCookie = readCookie(BRANDING_COOKIE);
  if (fromCookie) {
    cached = parseBrandingCookie(fromCookie);
  } else {
    cached = migrateLegacyStorage() ?? DEFAULT_BRANDING;
  }
  return cached;
}

export function getBrandingSnapshot(): SystemBrandingConfig {
  return read();
}

// Fallback de server snapshot para contextos sem o cookie (ex.: testes que
// não montam o BrandingProvider). O BrandingProvider passa o seu próprio,
// derivado do cookie lido no layout.
export function getBrandingServerSnapshot(): SystemBrandingConfig {
  return DEFAULT_BRANDING;
}

export function subscribeBranding(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function emit() {
  for (const l of listeners) l();
}

export function updateBrandingStore(partial: Partial<SystemBrandingConfig>): void {
  cached = { ...read(), ...partial };
  writeCookie(BRANDING_COOKIE, serializeBrandingCookie(cached));
  emit();
}

export function resetBrandingStore(): void {
  cached = DEFAULT_BRANDING;
  deleteCookie(BRANDING_COOKIE);
  try {
    window.localStorage.removeItem(LEGACY_STORAGE_KEY);
  } catch {
    // ignora
  }
  emit();
}
