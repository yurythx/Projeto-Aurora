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
        <Topbar userLabel="ana@projeto-nova.local" connectionState="open" onToggleSidebar={() => {}} />
      </BrandingProvider>
    );
    expect(screen.getByTestId("user-menu-stub")).toHaveTextContent("ana@projeto-nova.local");
  });
});
