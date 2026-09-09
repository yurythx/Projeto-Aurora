"use client";

import { Suspense, useState, useSyncExternalStore, type ReactNode } from "react";

import { Sidebar } from "@/components/layout/Sidebar";
import { Topbar } from "@/components/layout/Topbar";
import { Footer } from "@/components/layout/Footer";
import { ConnectionStateProvider } from "@/components/layout/ConnectionStateContext";
import { AuthFlashToast } from "@/components/layout/AuthFlashToast";
import { LGPDConsentModal } from "@/components/layout/LGPDConsentModal";
import { NotificationCenter } from "@/components/notifications/NotificationCenter";
import { NotificationHistoryProvider } from "@/components/notifications/NotificationHistoryProvider";
import { ToastProvider } from "@/components/notifications/ToastProvider";
import {
  getSidebarCollapsedSnapshot,
  setSidebarCollapsed,
  subscribeSidebarCollapsed,
} from "@/lib/layout/sidebarCollapsedStore";
import type { ConnectionState } from "@/lib/websocket/client";

// Mesmo valor do breakpoint md: do Tailwind — usado para decidir se
// onToggleSidebar deve recolher (desktop) ou abrir/fechar (mobile) a
// Sidebar, já que os dois compartilham um único botão na Topbar.
const MD_BREAKPOINT_QUERY = "(min-width: 768px)";

// Layout raiz do dashboard: Sidebar recolhível + Topbar fixos (a Topbar
// inclui a barra e-MAG de acessibilidade), conteúdo da rota no meio, e o
// NotificationCenter (invisível, só lógica) montado uma vez aqui para
// alimentar o indicador de conexão da Topbar, a pilha de toasts e a
// bandeja do sino.
//
// O BrandingProvider e o VLibrasWidget NÃO ficam mais aqui — subiram para
// app/providers.tsx (raiz), para valerem também nas páginas públicas
// (/, /sobre, /acessibilidade, /login).
//
// Âncoras dos atalhos e-MAG (ver EMagAccessibilityBar): #conteudo (Alt+1),
// #menu (Alt+2), #rodape (Alt+4).

export function DashboardShell({
  userLabel,
  initialTheme,
  initialCollapsed = false,
  children,
}: {
  userLabel: string;
  initialTheme?: "light" | "dark";
  /** Estado da Sidebar lido do cookie `nova-sidebar-collapsed` no layout do
   * servidor — server snapshot do useSyncExternalStore, para o shell já
   * nascer recolhido/expandido no 1º paint sem "piscar" na hidratação. */
  initialCollapsed?: boolean;
  children: ReactNode;
}) {
  const [connectionState, setConnectionState] = useState<ConnectionState>("idle");
  const collapsed = useSyncExternalStore(
    subscribeSidebarCollapsed,
    getSidebarCollapsedSnapshot,
    () => initialCollapsed,
  );
  const [mobileOpen, setMobileOpen] = useState(false);

  function toggleSidebar() {
    if (window.matchMedia(MD_BREAKPOINT_QUERY).matches) {
      setSidebarCollapsed(!collapsed);
    } else {
      setMobileOpen((prev) => !prev);
    }
  }

  return (
    <ToastProvider>
      <NotificationHistoryProvider>
        <Suspense fallback={null}>
          <AuthFlashToast />
        </Suspense>
        <a
          href="#conteudo"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[60] focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-primary-foreground"
        >
          Pular para o conteúdo [1]
        </a>

        <Topbar
          userLabel={userLabel}
          connectionState={connectionState}
          onToggleSidebar={toggleSidebar}
          sidebarExpanded={mobileOpen || !collapsed}
          initialTheme={initialTheme}
        />
        <div id="menu">
          <Sidebar
            collapsed={collapsed}
            mobileOpen={mobileOpen}
            onCloseMobile={() => setMobileOpen(false)}
          />
        </div>

        {/* Offset do shell: pt = altura da Topbar, pl (md+) = largura EXATA
            da Sidebar — assim o conteúdo e o rodapé encostam na Sidebar sem
            faixa morta. O respiro lateral vem do px do <main>/<Footer>, não
            de um pl extra. min-h-dvh (não screen) mantém o rodapé colado no
            fim da viewport em telas curtas, sem sobra. */}
        <div
          className={`flex min-h-dvh flex-col pt-[var(--topbar-h)] transition-[padding] duration-[var(--shell-motion)] ease-[var(--shell-ease)]
            ${collapsed ? "md:pl-[var(--sidebar-w-collapsed)]" : "md:pl-[var(--sidebar-w)]"}`}
        >
          {/* tabIndex={-1}: torna o alvo de "Pular para o conteúdo" focável de
              forma consistente entre navegadores (A-13). */}
          <main
            id="conteudo"
            tabIndex={-1}
            className="flex-1 overflow-x-auto px-4 pb-8 outline-none sm:px-8 sm:pb-10"
          >
            <ConnectionStateProvider value={connectionState}>{children}</ConnectionStateProvider>
          </main>
          <div id="rodape">
            <Footer />
          </div>
        </div>

        <NotificationCenter onConnectionStateChange={setConnectionState} />
        <LGPDConsentModal />
      </NotificationHistoryProvider>
    </ToastProvider>
  );
}
