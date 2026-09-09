"use client";

import { Suspense, type ReactNode } from "react";
import Link from "next/link";
import { LogOut } from "lucide-react";
import { useSession } from "next-auth/react";

import { EMagAccessibilityBar } from "@/components/layout/EMagAccessibilityBar";
import { LGPDConsentModal } from "@/components/layout/LGPDConsentModal";
import { Footer } from "@/components/layout/Footer";
import { Logo } from "@/components/ui/Logo";
import { Button } from "@/components/ui/Button";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { useBranding } from "@/components/branding/BrandingContext";
import { fullSignOut } from "@/lib/auth/logout";
import { ToastProvider } from "@/components/notifications/ToastProvider";
import { AuthFlashToast } from "@/components/layout/AuthFlashToast";

// Shell das páginas PÚBLICAS (fora da área autenticada) — mesma barra e-MAG
// fixa da área interna, header institucional enxuto e o rodapé compartilhado.
// Os âncoras #menu / #conteudo / #rodape são os alvos dos atalhos e-MAG
// Alt+2 / Alt+1 / Alt+4 (ver EMagAccessibilityBar).
export function PublicShell({ children }: { children: ReactNode }) {
  const { branding } = useBranding();
  const { status } = useSession();
  const authenticated = status === "authenticated";

  return (
    <ToastProvider>
    <div className="flex min-h-screen flex-col pt-[var(--topbar-h)]">
      {/* Sem Suspense, useSearchParams() dentro de AuthFlashToast faria o
          Next.js reclamar em build/desligar a otimização estática desta
          página inteira — o fallback null é invisível de qualquer jeito
          (o componente não renderiza nada). */}
      <Suspense fallback={null}>
        <AuthFlashToast />
      </Suspense>
      <header
        id="menu"
        className="fixed inset-x-0 top-0 z-50 flex h-[var(--topbar-h)] flex-col border-b border-surface-border bg-surface shadow-xs"
      >
        <EMagAccessibilityBar />
        <div className="mx-auto flex w-full max-w-5xl flex-1 items-center justify-between px-6">
          <Link
            href={authenticated ? "/dashboard" : "/"}
            className="flex items-center gap-2.5 text-lg font-semibold hover:opacity-90 transition-opacity"
          >
            <Logo size={30} />
            <span className="font-bold text-foreground">{branding.appName}</span>
          </Link>
          <nav className="flex items-center gap-4 text-sm">
            <ThemeToggle />
            <Link href="/sobre" className="text-muted hover:text-foreground transition-colors">
              Sobre
            </Link>
            <Link
              href="/padroes"
              className="hidden text-muted hover:text-foreground transition-colors sm:inline"
            >
              Padrões
            </Link>
            {authenticated ? (
              <>
                <Link
                  href="/dashboard"
                  className="text-muted hover:text-foreground transition-colors"
                >
                  Painel
                </Link>
                <button
                  type="button"
                  onClick={() => void fullSignOut()}
                  aria-label="Sair da conta"
                  title="Sair"
                  className="inline-flex h-9 w-9 items-center justify-center rounded-md text-muted transition-colors hover:bg-black/5 hover:text-danger dark:hover:bg-white/5"
                >
                  <LogOut size={18} aria-hidden="true" />
                </button>
              </>
            ) : (
              <Link href="/login">
                <Button size="sm">Entrar</Button>
              </Link>
            )}
          </nav>
        </div>
      </header>

      {/* tabIndex={-1}: torna o alvo do "Pular para o conteúdo" focável de
          forma consistente entre navegadores (A-13). */}
      <main id="conteudo" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </main>

      <div id="rodape" tabIndex={-1}>
        <Footer />
      </div>
      <LGPDConsentModal />
    </div>
    </ToastProvider>
  );
}
