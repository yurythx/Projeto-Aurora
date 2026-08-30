import type { Metadata } from "next";
import { Geist_Mono, Inter, Newsreader } from "next/font/google";
import { cookies } from "next/headers";
import type { ReactNode } from "react";
import "./globals.css";

import { BRANDING_COOKIE, parseBrandingCookie } from "@/components/branding/brandingConfig";
import { Providers } from "./providers";

// Par tipográfico deliberado (§ redesenho 2026-08, ver globals.css):
// Newsreader (serifada de notícia) só nos títulos, Inter no corpo,
// Geist Mono reservada a códigos. Os três expõem custom properties que
// o @theme inline do globals.css consome.
const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
});

const newsreader = Newsreader({
  variable: "--font-newsreader",
  subsets: ["latin"],
  weight: ["500", "600"],
  style: ["normal", "italic"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Projeto Nova",
  description: "Plataforma de fiscalização de contratos municipais e inteligência do Diário Oficial de Rondonópolis.",
};

export default async function RootLayout({ children }: { children: ReactNode }) {
  // Lê o cookie "nova-theme" (escrito por components/ui/ThemeToggle.tsx)
  // no servidor e carimba data-theme em <html> ANTES do primeiro paint —
  // zero flash de tema errado, sem precisar de nenhum <script> inline
  // (que a CSP com nonce deste app bloquearia — ver a nota em
  // ThemeToggle.tsx). Sem cookie (primeira visita), nenhum atributo é
  // definido e o CSS puro em globals.css segue prefers-color-scheme.
  const cookieStore = await cookies();
  const theme = cookieStore.get("nova-theme")?.value;
  const dataTheme = theme === "dark" || theme === "light" ? theme : undefined;

  // Alto Contraste e-MAG também via cookie: sem isto o atributo só era
  // aplicado por um useEffect no BrandingProvider, então a página piscava
  // do contraste normal para o alto a cada refresh. Mesmo cookie que a
  // Topbar escreve ao alternar (components/branding/brandingStore.ts).
  const dataHighContrast = parseBrandingCookie(cookieStore.get(BRANDING_COOKIE)?.value).highContrast
    ? "true"
    : undefined;

  return (
    <html
      lang="pt-BR"
      data-theme={dataTheme}
      data-high-contrast={dataHighContrast}
      className={`${inter.variable} ${newsreader.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
