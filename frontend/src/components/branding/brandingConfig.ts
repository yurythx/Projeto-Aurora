// Formato + padrões do branding white-label do Projeto Aurora.

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
  appName: "Projeto Aurora",
  appDescription: "Plataforma Enterprise Base Genérica & Serviços",
  orgName: "Prefeitura Municipal de Rondonópolis",
  logoUrl: "",
  faviconUrl: "",
  supportEmail: "suporte@rondonopolis.mt.gov.br",
  supportPhone: "(66) 3411-5000",
  supportHours: "Segunda a Sexta, das 08h às 17h",
  highContrast: false,
};

export const BRANDING_COOKIE = "aurora-branding";

export function parseBrandingCookie(raw: string | null | undefined): SystemBrandingConfig {
  if (!raw) return DEFAULT_BRANDING;
  const attempts = [raw];
  if (raw.includes("%")) {
    try {
      attempts.push(decodeURIComponent(raw));
    } catch {
      // valor malformado
    }
  }
  for (const text of attempts) {
    try {
      const parsed = JSON.parse(text) as Partial<SystemBrandingConfig> | null;
      if (parsed && typeof parsed === "object") {
        return { ...DEFAULT_BRANDING, ...parsed };
      }
    } catch {
      // tenta próximo
    }
  }
  return DEFAULT_BRANDING;
}

export function serializeBrandingCookie(config: SystemBrandingConfig): string {
  return JSON.stringify(config);
}
