"use client";

import React, { createContext, useContext, useEffect, useState } from "react";

export interface SystemBrandingConfig {
  appName: string;
  appDescription: string;
  orgName: string;
  logoUrl: string;
  faviconUrl: string;
  supportEmail: string;
  supportPhone: string;
  supportHours: string;
  highContrast: boolean;
}

export const DEFAULT_BRANDING: SystemBrandingConfig = {
  appName: "Projeto Nova",
  appDescription: "Sistema de Gestão de Contratos Administrativos e Liquidação Financeira",
  orgName: "Prefeitura Municipal de Rondonópolis",
  logoUrl: "",
  faviconUrl: "",
  supportEmail: "suporte.contratos@rondonopolis.mt.gov.br",
  supportPhone: "(66) 3411-5000",
  supportHours: "Segunda a Sexta, das 08h às 17h",
  highContrast: false,
};

const BRANDING_STORAGE_KEY = "nova_system_branding_v1";

interface BrandingContextType {
  branding: SystemBrandingConfig;
  updateBranding: (newConfig: Partial<SystemBrandingConfig>) => void;
  resetBranding: () => void;
  toggleHighContrast: () => void;
}

const BrandingContext = createContext<BrandingContextType | undefined>(undefined);

export function BrandingProvider({ children }: { children: React.ReactNode }) {
  const [branding, setBranding] = useState<SystemBrandingConfig>(DEFAULT_BRANDING);

  // Carrega configurações salvas do localStorage no cliente
  useEffect(() => {
    try {
      const saved = localStorage.getItem(BRANDING_STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        setBranding((prev) => ({ ...prev, ...parsed }));
      }
    } catch {
      // fallback para os padrões
    }
  }, []);

  // Aplica o atributo e-MAG de Alto Contraste no elemento <html>
  useEffect(() => {
    if (typeof document !== "undefined") {
      const root = document.documentElement;
      if (branding.highContrast) {
        root.setAttribute("data-high-contrast", "true");
      } else {
        root.removeAttribute("data-high-contrast");
      }
    }
  }, [branding.highContrast]);

  const updateBranding = (newConfig: Partial<SystemBrandingConfig>) => {
    setBranding((prev) => {
      const updated = { ...prev, ...newConfig };
      try {
        localStorage.setItem(BRANDING_STORAGE_KEY, JSON.stringify(updated));
      } catch {
        // ignora erro de armazenamento
      }
      return updated;
    });
  };

  const resetBranding = () => {
    setBranding(DEFAULT_BRANDING);
    try {
      localStorage.removeItem(BRANDING_STORAGE_KEY);
    } catch {
      // ignora
    }
  };

  const toggleHighContrast = () => {
    updateBranding({ highContrast: !branding.highContrast });
  };

  return (
    <BrandingContext.Provider value={{ branding, updateBranding, resetBranding, toggleHighContrast }}>
      {children}
    </BrandingContext.Provider>
  );
}

export function useBranding() {
  const context = useContext(BrandingContext);
  if (!context) {
    throw new Error("useBranding deve ser utilizado dentro de um BrandingProvider");
  }
  return context;
}
