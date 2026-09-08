"use client";

import { LayoutDashboard, Plug, Settings, Activity, Box } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";

import useSWR from "swr";
import { apiClient } from "@/lib/api/client";
import { useBranding } from "@/components/branding/BrandingContext";
import type { FeatureFlag } from "@/types/api";
import packageJson from "../../../package.json";

// Versão do próprio frontend (package.json) — só informativa, mostrada no
// rodapé da barra lateral (ver o retângulo abaixo). Não é a mesma coisa
// que uma versão de API/release do sistema como um todo (este projeto não
// tem esse conceito hoje), mas já ajuda a identificar o build em suporte
// ("qual versão você está vendo?") sem precisar abrir o DevTools.
const FRONTEND_VERSION = packageJson.version;

const links: { href: string; label: string; icon: typeof LayoutDashboard; flag?: string; match?: string[] }[] = [
  { href: "/dashboard", label: "Visão Geral", icon: LayoutDashboard },
  { href: "/exemplos", label: "Módulo Modelo", icon: Box, flag: "module_exemplos_enabled" },
  { href: "/monitoramento", label: "Monitoramento", icon: Activity },
  { href: "/integracoes", label: "Integrações", icon: Plug },
  { href: "/configuracao", label: "Configurações", icon: Settings },
];

function matchesPath(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(prefix + "/");
}

interface SidebarProps {
  collapsed: boolean;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

export function Sidebar({ collapsed, mobileOpen, onCloseMobile }: SidebarProps) {
  const pathname = usePathname();
  const { branding } = useBranding();

  const { data: featureFlags } = useSWR<FeatureFlag[]>(
    "v1/admin/feature-flags",
    () => apiClient.get<FeatureFlag[]>("v1/admin/feature-flags").then((res) => res.data),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );

  const disabledFlags = new Set(
    (featureFlags ?? []).filter((f) => f.enabled === false).map((f) => f.key)
  );

  const visibleLinks = links.filter((l) => !l.flag || !disabledFlags.has(l.flag));

  let activeHref: string | null = null;
  let bestLen = -1;
  for (const l of visibleLinks) {
    for (const prefix of l.match ?? [l.href]) {
      if (matchesPath(pathname, prefix) && prefix.length > bestLen) {
        bestLen = prefix.length;
        activeHref = l.href;
      }
    }
  }

  useEffect(() => {
    if (!mobileOpen) return;
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onCloseMobile();
    }
    document.addEventListener("keydown", onKeyDown);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previousOverflow;
    };
  }, [mobileOpen, onCloseMobile]);

  return (
    <>
      {mobileOpen && (
        <div
          className="fixed inset-x-0 bottom-0 top-[var(--topbar-h)] z-40 bg-black/40 md:hidden"
          onClick={onCloseMobile}
          aria-hidden="true"
        />
      )}

      <nav
        aria-label="Principal"
        className={`fixed left-0 bottom-0 top-[var(--topbar-h)] z-40 flex flex-col overflow-y-auto overflow-x-hidden border-r border-surface-border bg-surface
          transition-[transform,width] duration-[var(--shell-motion)] ease-[var(--shell-ease)] md:translate-x-0
          ${collapsed ? "md:w-[var(--sidebar-w-collapsed)]" : "md:w-[var(--sidebar-w)]"}
          ${mobileOpen ? "translate-x-0" : "-translate-x-full"} w-72`}
      >
        {!collapsed && (
          <div className="px-3 pt-4 pb-1 text-xs font-semibold uppercase tracking-widest text-muted">
            Menu Principal
          </div>
        )}
        <ul className={`flex flex-1 flex-col gap-0.5 px-2 pb-3 ${collapsed ? "pt-3" : "pt-2"}`}>
          {visibleLinks.map((link) => {
            const Icon = link.icon;
            const active = link.href === activeHref;
            return (
              <li key={link.href}>
                <Link
                  href={link.href}
                  onClick={onCloseMobile}
                  title={collapsed ? link.label : undefined}
                  // A-15: com a Sidebar recolhida no desktop, o <span> do
                  // rótulo vira `md:hidden` (removido da árvore de
                  // acessibilidade) e o ícone é aria-hidden — sem
                  // aria-label o link fica sem NENHUM nome acessível
                  // (title não é uma fonte confiável de nome acessível ao
                  // navegar por teclado/leitor de tela).
                  aria-label={collapsed ? link.label : undefined}
                  aria-current={active ? "page" : undefined}
                  className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors
                    ${active ? "bg-primary/10 text-primary" : "text-foreground hover:bg-black/5 dark:hover:bg-white/5"}
                    ${collapsed ? "md:justify-center" : ""}`}
                >
                  <Icon size={18} aria-hidden="true" className="shrink-0" />
                  <span className={collapsed ? "md:hidden" : ""}>{link.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>

        {/* Rodapé informativo — NUNCA identidade/logout do usuário: esses
            continuam só no menu do usuário na Topbar (mesmo padrão do
            GovBR-DS/gov.br/SEI! — cabeçalho concentra conta+sessão, a
            barra lateral é navegação pura). Só nome do sistema + versão
            do build, útil pra suporte ("qual versão você está vendo?"). */}
        <div
          className={`mt-auto shrink-0 border-t border-surface-border px-3 py-3 text-[11px] leading-tight text-muted
            ${collapsed ? "md:px-0 md:text-center" : ""}`}
          title={`${branding.appName} · v${FRONTEND_VERSION}`}
        >
          <p className={`truncate font-medium text-foreground/70 ${collapsed ? "md:hidden" : ""}`}>
            {branding.appName}
          </p>
          <p className={collapsed ? "md:hidden" : ""}>v{FRONTEND_VERSION}</p>
        </div>
      </nav>
    </>
  );
}
