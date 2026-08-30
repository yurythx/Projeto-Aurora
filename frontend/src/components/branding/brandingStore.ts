"use client";

import { DEFAULT_BRANDING, type SystemBrandingConfig } from "./brandingConfig";

// "External store" (no sentido de useSyncExternalStore) do branding
// white-label persistido em localStorage — mesmo raciocínio de
// lib/layout/sidebarCollapsedStore.ts: ler localStorage e então chamar
// setState num useEffect é o anti-padrão que react-hooks/set-state-in-effect
// desencoraja. useSyncExternalStore lê o snapshot de servidor
// (DEFAULT_BRANDING) no SSR/1º paint e troca pelo do cliente após a
// hidratação, sem mismatch e sem efeito.

const STORAGE_KEY = "nova_system_branding_v1";

type Listener = () => void;
const listeners = new Set<Listener>();
let cached: SystemBrandingConfig | null = null;

function read(): SystemBrandingConfig {
  if (cached) return cached;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    cached = raw
      ? { ...DEFAULT_BRANDING, ...(JSON.parse(raw) as Partial<SystemBrandingConfig>) }
      : DEFAULT_BRANDING;
  } catch {
    cached = DEFAULT_BRANDING;
  }
  return cached;
}

export function getBrandingSnapshot(): SystemBrandingConfig {
  return read();
}

// O servidor nunca teve um branding do dispositivo — usa os padrões.
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
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cached));
  } catch {
    // Sem persistência nesta sessão (aba privada / política do navegador)
    // — a UI ainda reflete a escolha até recarregar.
  }
  emit();
}

export function resetBrandingStore(): void {
  cached = DEFAULT_BRANDING;
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignora
  }
  emit();
}
