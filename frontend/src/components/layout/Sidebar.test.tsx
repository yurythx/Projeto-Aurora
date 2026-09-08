import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

const { usePathname } = vi.hoisted(() => ({ usePathname: vi.fn() }));
vi.mock("next/navigation", () => ({ usePathname }));

import { Sidebar } from "./Sidebar";
import { BrandingProvider, DEFAULT_BRANDING } from "@/components/branding/BrandingContext";

// useBranding() (usado pelo rodapé informativo — nome do sistema + versão)
// exige um BrandingProvider como ancestral; ver o mesmo padrão em
// Topbar.test.tsx.
function renderSidebar(props: React.ComponentProps<typeof Sidebar>) {
  return render(
    <BrandingProvider>
      <Sidebar {...props} />
    </BrandingProvider>,
  );
}

describe("Sidebar", () => {
  it("marca só o link exato como ativo em /dashboard", () => {
    usePathname.mockReturnValue("/dashboard");
    renderSidebar({ collapsed: false, mobileOpen: false, onCloseMobile: () => {} });
    expect(screen.getByRole("link", { name: /Visão Geral/ })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("link", { name: /Integrações/ })).not.toHaveAttribute("aria-current");
  });

  it("acende a seção 'Integrações' na rota de integrações", () => {
    usePathname.mockReturnValue("/integracoes");
    renderSidebar({ collapsed: false, mobileOpen: false, onCloseMobile: () => {} });
    expect(screen.getByRole("link", { name: /Integrações/ })).toHaveAttribute("aria-current", "page");
  });

  it("clicar num link fecha o painel mobile (onCloseMobile)", async () => {
    usePathname.mockReturnValue("/dashboard");
    const user = userEvent.setup();
    const onCloseMobile = vi.fn();
    renderSidebar({ collapsed: false, mobileOpen: true, onCloseMobile });

    await user.click(screen.getByRole("link", { name: /Integrações/ }));
    expect(onCloseMobile).toHaveBeenCalled();
  });

  it("Escape fecha o painel mobile só quando ele está aberto", async () => {
    usePathname.mockReturnValue("/dashboard");
    const user = userEvent.setup();
    const onCloseMobile = vi.fn();
    const { rerender } = render(
      <BrandingProvider>
        <Sidebar collapsed={false} mobileOpen={false} onCloseMobile={onCloseMobile} />
      </BrandingProvider>,
    );

    await user.keyboard("{Escape}");
    expect(onCloseMobile).not.toHaveBeenCalled();

    rerender(
      <BrandingProvider>
        <Sidebar collapsed={false} mobileOpen onCloseMobile={onCloseMobile} />
      </BrandingProvider>,
    );
    await user.keyboard("{Escape}");
    expect(onCloseMobile).toHaveBeenCalled();
  });

  it("clicar no overlay do painel mobile fecha o painel", async () => {
    usePathname.mockReturnValue("/dashboard");
    const user = userEvent.setup();
    const onCloseMobile = vi.fn();
    const { container } = renderSidebar({ collapsed: false, mobileOpen: true, onCloseMobile });

    const overlay = container.querySelector('[aria-hidden="true"].fixed');
    expect(overlay).not.toBeNull();
    await user.click(overlay as Element);
    expect(onCloseMobile).toHaveBeenCalled();
  });

  // Rodapé informativo (achado de consistência com a página de
  // Acessibilidade/GovBR-DS): nome do sistema + versão do build, nunca
  // identidade/logout do usuário — esses ficam só no menu do usuário na
  // Topbar (§ padrão gov.br/SEI!).
  it("mostra o nome do sistema e a versão no rodapé, sem nenhum controle de logout", () => {
    usePathname.mockReturnValue("/dashboard");
    renderSidebar({ collapsed: false, mobileOpen: false, onCloseMobile: () => {} });

    expect(screen.getByText(DEFAULT_BRANDING.appName)).toBeInTheDocument();
    expect(screen.getByText(/^v\d+\.\d+\.\d+$/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sair/i })).not.toBeInTheDocument();
  });
});
