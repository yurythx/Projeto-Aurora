"use client";

import React, { createContext, useContext, useEffect, useSyncExternalStore } from "react";

import { DEFAULT_BRANDING, type SystemBrandingConfig } from "./brandingConfig";
import {
  getBrandingSnapshot,
  resetBrandingStore,
  subscribeBranding,
  updateBrandingStore,
} from "./brandingStore";
import { safeResourceUrl } from "@/lib/security/safe-url";

// Re-export para não quebrar imports antigos (BrandingSettingsForm etc.).
export { DEFAULT_BRANDING };
export type { SystemBrandingConfig };

interface BrandingContextType {
  branding: SystemBrandingConfig;
  updateBranding: (newConfig: Partial<SystemBrandingConfig>) => void;
  resetBranding: () => void;
  toggleHighContrast: () => void;
  increaseFontSize: () => void;
  decreaseFontSize: () => void;
  resetFontSize: () => void;
}

const BrandingContext = createContext<BrandingContextType | undefined>(undefined);

export function BrandingProvider({
  children,
  initialBranding,
}: {
  children: React.ReactNode;
  /** Branding lido do cookie `nova-branding` no layout do servidor. Vira o
   * server snapshot do useSyncExternalStore: como o cliente lê o MESMO
   * cookie, o snapshot de SSR e o do cliente coincidem e o nome não pisca
   * do valor antigo para o novo depois da hidratação. Ausente (fora do
   * layout, ex. testes) → DEFAULT_BRANDING. */
  initialBranding?: SystemBrandingConfig;
}) {
  // Estado vive fora do React (cookie) — useSyncExternalStore em vez de
  // useState + useEffect de hidratação. Ver brandingStore.ts.
  const serverSnapshot = initialBranding ?? DEFAULT_BRANDING;
  const branding = useSyncExternalStore(
    subscribeBranding,
    getBrandingSnapshot,
    () => serverSnapshot,
  );

  // Sincroniza os atributos e-MAG (Alto Contraste + escala tipográfica) no
  // <html> — uso legítimo de useEffect (sincronizar com um sistema externo,
  // o DOM), sem chamar setState.
  useEffect(() => {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    if (branding.highContrast) {
      root.setAttribute("data-high-contrast", "true");
    } else {
      root.removeAttribute("data-high-contrast");
    }
    root.setAttribute("data-font-scale", String(branding.fontSizeScale || 100));
  }, [branding.highContrast, branding.fontSizeScale]);

  // Favicon white-label: branding.faviconUrl era um campo declarado desde
  // sempre (tipo + default + persistido no cookie), mas nada o lia — o
  // <link rel="icon"> ficava sempre no favicon padrão do Projeto Aurora
  // (achado de auditoria). safeResourceUrl (S-03) aplica a mesma
  // allowlist de esquema que a logo já usa: só https:// vira href de
  // verdade. marca um data-attribute próprio no <link> pra saber, na
  // próxima rodada, se foi ESTE efeito que o escreveu (nunca mexe num
  // <link rel="icon"> que já estivesse lá por outro motivo antes do
  // primeiro branding customizado, e volta ao arquivo estático de
  // app/icon.svg/favicon.ico assim que a URL é removida/fica inválida).
  useEffect(() => {
    if (typeof document === "undefined") return;
    const safeFavicon = safeResourceUrl(branding.faviconUrl);
    const marker = "data-aurora-branding-favicon";
    let link = document.querySelector<HTMLLinkElement>(`link[rel="icon"][${marker}]`);

    if (!safeFavicon) {
      link?.remove();
      return;
    }
    if (!link) {
      link = document.createElement("link");
      link.rel = "icon";
      link.setAttribute(marker, "true");
      document.head.appendChild(link);
    }
    link.href = safeFavicon;
  }, [branding.faviconUrl]);

  const increaseFontSize = () => {
    const current = branding.fontSizeScale || 100;
    if (current < 130) updateBrandingStore({ fontSizeScale: current + 10 });
  };

  const decreaseFontSize = () => {
    const current = branding.fontSizeScale || 100;
    if (current > 90) updateBrandingStore({ fontSizeScale: current - 10 });
  };

  const resetFontSize = () => updateBrandingStore({ fontSizeScale: 100 });

  const value: BrandingContextType = {
    branding,
    updateBranding: updateBrandingStore,
    resetBranding: resetBrandingStore,
    toggleHighContrast: () => updateBrandingStore({ highContrast: !branding.highContrast }),
    increaseFontSize,
    decreaseFontSize,
    resetFontSize,
  };

  return <BrandingContext.Provider value={value}>{children}</BrandingContext.Provider>;
}

export function useBranding() {
  const context = useContext(BrandingContext);
  if (!context) {
    throw new Error("useBranding deve ser utilizado dentro de um BrandingProvider");
  }
  return context;
}
