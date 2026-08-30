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

// Cookie (não localStorage) para o branding chegar ao servidor e o layout
// já renderizar a Topbar/Footer com o nome certo no 1º paint — ver
// lib/prefs/cookies.ts. O valor é o JSON do SystemBrandingConfig.
export const BRANDING_COOKIE = "nova-branding";

// parseBrandingCookie é isomórfico (sem window/document): o layout do
// servidor e a store do cliente usam o MESMO parser, garantindo que o
// snapshot de SSR e o do cliente coincidam — sem essa igualdade, o
// useSyncExternalStore troca o valor logo após a hidratação e pisca.
export function parseBrandingCookie(raw: string | null | undefined): SystemBrandingConfig {
  if (!raw) return DEFAULT_BRANDING;
  const attempts = [raw];
  // Rede de segurança: se o valor chegar ainda percent-encoded (algum
  // runtime não decodifica o cookie), tenta decodificar uma vez.
  if (raw.includes("%")) {
    try {
      attempts.push(decodeURIComponent(raw));
    } catch {
      // valor malformado — cai no DEFAULT abaixo
    }
  }
  for (const text of attempts) {
    try {
      const parsed = JSON.parse(text) as Partial<SystemBrandingConfig> | null;
      if (parsed && typeof parsed === "object") {
        return { ...DEFAULT_BRANDING, ...parsed };
      }
    } catch {
      // tenta o próximo candidato
    }
  }
  return DEFAULT_BRANDING;
}

export function serializeBrandingCookie(config: SystemBrandingConfig): string {
  return JSON.stringify(config);
}
