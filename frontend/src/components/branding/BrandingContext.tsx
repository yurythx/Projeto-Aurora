"use client";

import React, { createContext, useContext, useEffect, useSyncExternalStore } from "react";

import { DEFAULT_BRANDING, type SystemBrandingConfig } from "./brandingConfig";
import {
  getBrandingServerSnapshot,
  getBrandingSnapshot,
  resetBrandingStore,
  subscribeBranding,
  updateBrandingStore,
} from "./brandingStore";

// Re-export para não quebrar imports antigos (BrandingSettingsForm etc.).
export { DEFAULT_BRANDING };
export type { SystemBrandingConfig };

interface BrandingContextType {
  branding: SystemBrandingConfig;
  updateBranding: (newConfig: Partial<SystemBrandingConfig>) => void;
  resetBranding: () => void;
  toggleHighContrast: () => void;
}

const BrandingContext = createContext<BrandingContextType | undefined>(undefined);

export function BrandingProvider({ children }: { children: React.ReactNode }) {
  // Estado vive fora do React (localStorage) — useSyncExternalStore em vez
  // de useState + useEffect de hidratação. Ver brandingStore.ts.
  const branding = useSyncExternalStore(
    subscribeBranding,
    getBrandingSnapshot,
    getBrandingServerSnapshot,
  );

  // Sincroniza o atributo e-MAG de Alto Contraste no <html> — este SIM é
  // um uso legítimo de useEffect (sincronizar com um sistema externo, o
  // DOM), e não chama setState.
  useEffect(() => {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    if (branding.highContrast) {
      root.setAttribute("data-high-contrast", "true");
    } else {
      root.removeAttribute("data-high-contrast");
    }
  }, [branding.highContrast]);

  const value: BrandingContextType = {
    branding,
    updateBranding: updateBrandingStore,
    resetBranding: resetBrandingStore,
    toggleHighContrast: () => updateBrandingStore({ highContrast: !branding.highContrast }),
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
