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
  display: "swap",
});

export const metadata: Metadata = {
  title: "Projeto Aurora",
  description: "Plataforma Enterprise Base Genérica da Prefeitura Municipal de Rondonópolis.",
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

  // Branding lido server-side: Alto Contraste e escala de fonte e-MAG vão
  // como atributos em <html> ANTES do 1º paint (sem flash na hidratação), e
  // a config inteira desce como server snapshot do BrandingProvider. Mesmo
  // cookie que a barra e-MAG escreve (components/branding/brandingStore.ts).
  const initialBranding = parseBrandingCookie(cookieStore.get(BRANDING_COOKIE)?.value);
  const dataHighContrast = initialBranding.highContrast ? "true" : undefined;
  const dataFontScale = String(initialBranding.fontSizeScale || 100);

  return (
    <html
      lang="pt-BR"
      data-theme={dataTheme}
      data-high-contrast={dataHighContrast}
      data-font-scale={dataFontScale}
      className={`${inter.variable} ${newsreader.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <Providers initialBranding={initialBranding}>{children}</Providers>
      </body>
    </html>
  );
}
