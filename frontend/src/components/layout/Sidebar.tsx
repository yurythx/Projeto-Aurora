"use client";

import { LayoutDashboard, Plug, Settings, Activity, Box } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect } from "react";

const links: { href: string; label: string; icon: typeof LayoutDashboard; match?: string[] }[] = [
  { href: "/dashboard", label: "Visão Geral", icon: LayoutDashboard },
  { href: "/exemplos", label: "Módulo Modelo", icon: Box },
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
        <ul className={`flex flex-col gap-0.5 px-2 pb-3 ${collapsed ? "pt-3" : "pt-2"}`}>
          {links.map((link) => {
            const Icon = link.icon;
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
