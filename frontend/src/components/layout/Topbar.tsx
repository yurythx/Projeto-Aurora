"use client";

import { useState } from "react";
import { Menu } from "lucide-react";
import Link from "next/link";

import { NotificationBell } from "@/components/notifications/NotificationBell";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { UserMenu } from "@/components/layout/UserMenu";
import { EMagAccessibilityBar } from "@/components/layout/EMagAccessibilityBar";
import { Logo } from "@/components/ui/Logo";
import { useBranding } from "@/components/branding/BrandingContext";
import { safeResourceUrl } from "@/lib/security/safe-url";
import type { ConnectionState } from "@/lib/websocket/client";

const connectionCopy: Record<ConnectionState, { label: string; dotClass: string }> = {
  idle: { label: "Conectando…", dotClass: "bg-status-unknown" },
  connecting: { label: "Conectando…", dotClass: "bg-status-unknown" },
  open: { label: "Ao vivo", dotClass: "bg-status-online" },
  closed: { label: "Reconectando…", dotClass: "bg-status-degraded" },
  unauthorized: { label: "Sessão expirada", dotClass: "bg-status-offline" },
};

export function Topbar({
  userLabel,
  connectionState,
  onToggleSidebar,
  sidebarExpanded = false,
  onOpenGlobalSearch,
  initialTheme,
}: {
  userLabel: string;
  connectionState: ConnectionState;
  onToggleSidebar: () => void;
  sidebarExpanded?: boolean;
  onOpenGlobalSearch?: () => void;
  initialTheme?: "light" | "dark";
}) {
  const status = connectionCopy[connectionState];
  const { branding } = useBranding();
  const [logoError, setLogoError] = useState(false);
  // S-03: só usa a URL da logo se for https:// bem-formada.
  const logoSrc = logoError ? null : safeResourceUrl(branding.logoUrl);

  return (
    <header className="fixed inset-x-0 top-0 z-50 flex h-[var(--topbar-h)] flex-col border-b border-surface-border bg-surface/95 backdrop-blur-md shadow-xs transition-colors duration-200">
      {/* 1. Barra e-MAG de Acessibilidade Governamental Oficial (40px) */}
      <EMagAccessibilityBar onOpenGlobalSearch={onOpenGlobalSearch} />

      {/* 2. Topbar Principal de Navegação */}
      <div className="flex min-h-0 flex-1 items-center justify-between px-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onToggleSidebar}
            aria-label="Alternar menu lateral"
            aria-expanded={sidebarExpanded}
            aria-controls="menu"
            className="inline-flex h-10 w-10 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <Menu size={18} aria-hidden="true" />
          </button>

          {/* Logomarca Dinâmica com Fallback Vetorial Seguro */}
          <Link
            href="/dashboard"
            className="flex items-center gap-2.5 group focus:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-md p-1"
          >
            {logoSrc ? (
              // Espaço reservado (h-8 × 120px) para a logo white-label não
              // empurrar o layout ao carregar — CLS. URL externa
              // arbitrária, então next/image não se aplica.
              <span className="inline-flex h-8 w-[120px] shrink-0 items-center justify-start overflow-hidden">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={logoSrc}
                  alt={`Logomarca de ${branding.appName}`}
                  width={120}
                  height={32}
                  onError={() => setLogoError(true)}
                  className="h-8 w-auto max-w-full object-contain"
                />
              </span>
            ) : (
              // Mesmo selo institucional usado em /login, no rodapé e nas
              // páginas públicas (PublicShell) — sem uma logo branca
              // configurada, o dashboard mostrava um ShieldCheck genérico
              // em vez da marca de verdade, a única tela do sistema que
              // divergia desse padrão (achado de revisão de consistência).
              <Logo size={30} />
            )}

            <span className="flex flex-col">
              <span className="text-sm font-bold text-foreground leading-tight tracking-tight group-hover:text-primary transition-colors">
                {branding.appName}
              </span>
              <span className="text-[10px] text-muted font-mono uppercase tracking-[0.12em] leading-none">
                Plataforma Enterprise Base
              </span>
            </span>
          </Link>

          {/* aria-live: mudanças de conexão (Reconectando…, Sessão expirada)
              são anunciadas por leitor de tela (A-07). A cor nunca é o único
              indicador: há sempre texto (visível no md+, sr-only no mobile). */}
          <div
            className="flex items-center gap-2 text-xs text-muted md:ml-4 md:border-l md:border-surface-border md:pl-4"
            aria-live="polite"
          >
            <span
              className={`h-2 w-2 shrink-0 rounded-full transition-colors ${status.dotClass}`}
              aria-hidden="true"
            />
            <span className="hidden md:inline">{status.label}</span>
            <span className="sr-only md:hidden">Conexão: {status.label}</span>
          </div>

          {/* Mesmo link "Sobre" das páginas públicas (PublicShell) — a
              navegação de negócio continua vindo da Sidebar, este é só o
              link institucional que as demais telas do sistema já têm. */}
          <Link
            href="/sobre"
            className="hidden md:inline text-xs text-muted hover:text-foreground transition-colors md:ml-2"
          >
            Sobre
          </Link>
        </div>

        <div className="flex items-center gap-2">
          <ThemeToggle initialTheme={initialTheme} />
          <NotificationBell />
          <div className="ml-1">
            <UserMenu userLabel={userLabel} />
          </div>
        </div>
      </div>
    </header>
  );
}
