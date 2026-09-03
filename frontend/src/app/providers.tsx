"use client";

import { SessionProvider } from "next-auth/react";
import type { ReactNode } from "react";

import { BrandingProvider } from "@/components/branding/BrandingContext";
import type { SystemBrandingConfig } from "@/components/branding/brandingConfig";
import { VLibrasWidget } from "@/components/accessibility/VLibrasWidget";

// Provedores globais de TODA a aplicação (pública + autenticada):
// - SessionProvider (NextAuth): useSession()/signIn()/signOut() em qualquer
//   Client Component.
// - BrandingProvider (white-label + Alto Contraste e-MAG + escala de fonte):
//   fica na raiz, não no DashboardShell, para que a barra e-MAG e o
//   redimensionamento de fonte também valham nas páginas públicas
//   (/, /sobre, /acessibilidade, /login).
// - VLibrasWidget: widget oficial de tradução para Libras, presente em
//   toda página.
export function Providers({
  children,
  initialBranding,
}: {
  children: ReactNode;
  /** Branding lido do cookie `aurora-branding` no layout do servidor —
   * vira o server snapshot do useSyncExternalStore, evitando o flash do
   * nome/estado antigo na hidratação. */
  initialBranding?: SystemBrandingConfig;
}) {
  return (
    <SessionProvider>
      <BrandingProvider initialBranding={initialBranding}>
        {children}
        <VLibrasWidget />
      </BrandingProvider>
    </SessionProvider>
  );
}
