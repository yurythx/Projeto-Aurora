// Formato + padrões do branding white-label. Extraído de BrandingContext.tsx
// para que a "external store" (brandingStore.ts) possa importar sem ciclo.
// BrandingContext.tsx re-exporta os dois, então imports antigos seguem valendo.

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
