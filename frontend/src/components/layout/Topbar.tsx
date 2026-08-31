"use client";

import { useState } from "react";
import { Menu, Eye, Settings, ShieldCheck } from "lucide-react";
import Link from "next/link";

import { NotificationBell } from "@/components/notifications/NotificationBell";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { UserMenu } from "@/components/layout/UserMenu";
import { useBranding } from "@/components/branding/BrandingContext";
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
  initialTheme,
}: {
  userLabel: string;
  connectionState: ConnectionState;
  onToggleSidebar: () => void;
  initialTheme?: "light" | "dark";
}) {
  const status = connectionCopy[connectionState];
  const { branding, toggleHighContrast } = useBranding();
  const [logoError, setLogoError] = useState(false);

  return (
    <header className="fixed inset-x-0 top-0 z-50 flex h-[var(--topbar-h)] flex-col border-b border-surface-border bg-surface shadow-sm">
      {/* 1. Barra e-MAG de Acessibilidade Governamental Oficial (#14284B) */}
      <div className="flex h-7 shrink-0 items-center justify-between bg-header-topbar px-4 text-[11px] font-medium text-white/90">
        <div className="flex items-center gap-3">
          <span className="hidden sm:inline-block font-semibold uppercase tracking-wider text-white/80">
            {branding.orgName}
          </span>
          <span className="hidden md:inline text-white/40">•</span>
          <span className="truncate text-white/70">{branding.appDescription}</span>
        </div>

        {/* Controles e-MAG */}
        <div className="flex items-center gap-3 shrink-0">
          <button
            type="button"
            onClick={toggleHighContrast}
            aria-label="Alternar Modo Alto Contraste (Acessibilidade e-MAG)"
            className="flex items-center gap-1 hover:text-white transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent rounded px-1.5 py-0.5"
            title="Alternar Alto Contraste"
          >
            <Eye size={12} className={branding.highContrast ? "text-amber-300" : ""} />
            <span>{branding.highContrast ? "Alto Contraste: Ativo" : "Alto Contraste"}</span>
          </button>

          <Link
            href="/configuracao"
            aria-label="Configurações e Branding do Sistema"
            className="hidden sm:flex items-center gap-1 hover:text-white transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent rounded px-1.5 py-0.5"
          >
            <Settings size={12} />
            <span>Configurações</span>
          </Link>
        </div>
      </div>

      {/* 2. Topbar Principal de Navegação */}
      <div className="flex min-h-0 flex-1 items-center justify-between px-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onToggleSidebar}
            aria-label="Alternar menu lateral"
            className="inline-flex h-10 w-10 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <Menu size={18} aria-hidden="true" />
          </button>

          {/* Logomarca Dinâmica com Fallback Vetorial Seguro */}
          <Link href="/dashboard" className="flex items-center gap-2.5 group focus:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-md p-1">
            {branding.logoUrl && !logoError ? (
              // URL externa arbitrária (white-label); next/image exige domínio pré-configurado.
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={branding.logoUrl}
                alt={`Logomarca de ${branding.appName}`}
                onError={() => setLogoError(true)}
                className="h-8 w-auto max-w-[120px] object-contain"
              />
            ) : (
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold shadow-sm group-hover:bg-primary-hover transition-colors">
                <ShieldCheck size={20} />
              </div>
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

          <div
            className="hidden items-center gap-2 text-xs text-muted md:flex ml-4 pl-4 border-l border-surface-border"
            title="Status da conexão de notificações em tempo real"
          >
            <span
              className={`h-2 w-2 rounded-full transition-colors ${status.dotClass}`}
              aria-hidden="true"
            />
            <span>{status.label}</span>
          </div>
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
