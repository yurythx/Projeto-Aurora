import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

// NotificationBell/ThemeToggle/UserMenu têm sua própria lógica (e seus
// próprios testes) — mockados aqui como stubs pra isolar só o que é
// genuinamente do Topbar: o botão de alternar a Sidebar e o indicador de
// status da conexão WebSocket.
vi.mock("@/components/notifications/NotificationBell", () => ({
  NotificationBell: () => <div data-testid="notification-bell-stub" />,
}));
vi.mock("@/components/ui/ThemeToggle", () => ({
  ThemeToggle: () => <div data-testid="theme-toggle-stub" />,
}));
vi.mock("@/components/layout/UserMenu", () => ({
  UserMenu: ({ userLabel }: { userLabel: string }) => (
    <div data-testid="user-menu-stub">{userLabel}</div>
  ),
}));

import { Topbar } from "./Topbar";

import { BrandingProvider } from "@/components/branding/BrandingContext";

describe("Topbar", () => {
  it("clicar no botão de menu chama onToggleSidebar", async () => {
    const user = userEvent.setup();
    const onToggleSidebar = vi.fn();
    render(
      <BrandingProvider>
        <Topbar userLabel="admin" connectionState="open" onToggleSidebar={onToggleSidebar} />
      </BrandingProvider>
    );

    await user.click(screen.getByRole("button", { name: "Alternar menu lateral" }));
    expect(onToggleSidebar).toHaveBeenCalledTimes(1);
  });

  it.each([
    ["idle", "Conectando…"],
    ["connecting", "Conectando…"],
    ["open", "Ao vivo"],
    ["closed", "Reconectando…"],
    ["unauthorized", "Sessão expirada"],
  ] as const)("estado da conexão %s mostra o rótulo '%s'", (state, label) => {
    render(
      <BrandingProvider>
        <Topbar userLabel="admin" connectionState={state} onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("repassa userLabel pro UserMenu", () => {
    render(
      <BrandingProvider>
        <Topbar userLabel="ana@projeto-aurora.local" connectionState="open" onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.getByTestId("user-menu-stub")).toHaveTextContent("ana@projeto-aurora.local");
  });

  // Achado de consistência: sem uma logo branca configurada, o Topbar
  // mostrava um ShieldCheck genérico em vez do selo institucional
  // (components/ui/Logo) que /login, o rodapé e as páginas públicas
  // (PublicShell) já usavam — a única tela do sistema que divergia.
  it("sem logo branca configurada, mostra o mesmo selo institucional das páginas públicas", () => {
    render(
      <BrandingProvider>
        <Topbar userLabel="admin" connectionState="open" onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.getByRole("img", { name: "Projeto Aurora" })).toBeInTheDocument();
  });

  it("mostra o link Sobre, igual às páginas públicas", () => {
    render(
      <BrandingProvider>
        <Topbar userLabel="admin" connectionState="open" onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.getByRole("link", { name: "Sobre" })).toHaveAttribute("href", "/sobre");
  });

  // Não há mais um botão de logout duplicado na Topbar — só UM ponto de
  // saída (dentro do menu do usuário), igual ao padrão gov.br/GovBR-DS/SEI!
  // de manter identidade+logout exclusivamente no cabeçalho superior, sem
  // repetir o controle em dois lugares (achado de revisão de consistência).
  it("não duplica o botão de sair fora do menu do usuário", () => {
    render(
      <BrandingProvider>
        <Topbar userLabel="admin" connectionState="open" onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.queryByRole("button", { name: "Sair da conta" })).not.toBeInTheDocument();
  });
});
