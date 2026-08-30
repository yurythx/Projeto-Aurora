"use client";

import { LayoutDashboard, Newspaper, Plug, Settings, Bell, FileSignature } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";

// Navegação de 1º nível — uma entrada por SEÇÃO, não por página. As
// sub-páginas (Kanban↔Cadastro, Portal↔Busca↔Atos de Pessoal↔Revisão)
// ficam em abas dentro da seção (components/layout/SectionTabs). `match`
// lista os prefixos de rota que acendem a entrada — necessário quando a
// seção cobre caminhos que não são sub-rotas do próprio href (ex.:
// Diário Oficial cobre /diario e /pessoal, que são irmãos de
// /diario-oficial).
const links: { href: string; label: string; icon: typeof LayoutDashboard; match?: string[] }[] = [
  { href: "/dashboard", label: "Visão geral", icon: LayoutDashboard },
  { href: "/contratos", label: "Contratos", icon: FileSignature, match: ["/contratos"] },
  {
    href: "/diario-oficial",
    label: "Diário Oficial",
    icon: Newspaper,
    match: ["/diario-oficial", "/diario", "/pessoal"],
  },
  { href: "/monitoramento", label: "Monitoramento", icon: Bell },
  { href: "/integracoes", label: "Integrações", icon: Plug },
  { href: "/configuracao", label: "Configurações", icon: Settings },
];

function matchesPath(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(prefix + "/");
}

interface SidebarProps {
  /** Recolhida a ícone-somente no desktop (md: e acima) — persistida pelo
   * chamador (DashboardShell) em localStorage, uma preferência puramente
   * do dispositivo/navegador, não algo que precise sincronizar entre
   * abas ou dispositivos. */
  collapsed: boolean;
  /** Aberta como painel flutuante no mobile (abaixo de md:) — sempre
   * fechada por padrão a cada navegação, ao contrário de "collapsed". */
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

// Navegação lateral (§ Sidebar/Topbar no layout do papermoon.cloud):
// fixa, começando ABAIXO da Topbar (top-14 — a Topbar tem h-14 e também é
// fixa, ver Topbar.tsx) em vez de ao lado dela; recolhível a uma trilha de
// ícones no desktop, painel off-canvas com overlay no mobile. Sem cabeçalho
// próprio (logo/fechar) — a Topbar já cobre os dois (logo sempre visível,
// o mesmo botão de hambúrguer fecha o painel mobile).
export function Sidebar({ collapsed, mobileOpen, onCloseMobile }: SidebarProps) {
  const pathname = usePathname();

  // href da entrada cujo prefixo mais específico casa com a rota atual —
  // só essa acende. Cada entrada casa pelo próprio href e por qualquer
  // prefixo em `match`.
  let activeHref: string | null = null;
  let bestLen = -1;
  for (const l of links) {
    for (const prefix of l.match ?? [l.href]) {
      if (matchesPath(pathname, prefix) && prefix.length > bestLen) {
        bestLen = prefix.length;
        activeHref = l.href;
      }
    }
  }

  // Fecha o painel mobile com Escape, e trava o scroll do body enquanto
  // ele está aberto — o mesmo comportamento de qualquer off-canvas/modal.
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
      {/* Overlay do painel mobile — só existe (e só captura clique) com o
          painel aberto; invisível e fora do fluxo de layout no desktop. */}
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
          transition-[transform,width] duration-200 md:translate-x-0
          ${collapsed ? "md:w-[var(--sidebar-w-collapsed)]" : "md:w-[var(--sidebar-w)]"}
          ${mobileOpen ? "translate-x-0" : "-translate-x-full"} w-72`}
      >
        {!collapsed && (
          <div className="px-3 pt-4 pb-1 text-xs font-semibold uppercase tracking-widest text-muted">
            Principal
          </div>
        )}
        <ul className={`flex flex-col gap-0.5 px-2 pb-3 ${collapsed ? "pt-3" : "pt-2"}`}>
          {links.map((link) => {
            const Icon = link.icon;
            // Só UM item acende: o de href mais específico que casa com a
            // rota atual. Sem isso, em /diario/revisao os dois itens
            // (/diario e /diario/revisao) acendiam juntos, porque ambos
            // passam no startsWith. startsWith + "/" ainda cobre sub-rotas
            // sem item próprio (ex.: /contratos/demandas/... acende
            // "Liquidação").
            const active = link.href === activeHref;
            return (
              <li key={link.href}>
                <Link
                  href={link.href}
                  onClick={onCloseMobile}
                  title={collapsed ? link.label : undefined}
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
      </nav>
    </>
  );
}
