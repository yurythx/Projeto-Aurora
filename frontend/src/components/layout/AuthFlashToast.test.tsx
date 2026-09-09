import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const replace = vi.fn();
let searchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
  usePathname: () => "/",
  useSearchParams: () => searchParams,
}));

import { AuthFlashToast } from "./AuthFlashToast";
import { ToastProvider } from "@/components/notifications/ToastProvider";

function renderWithParams(query: string) {
  searchParams = new URLSearchParams(query);
  replace.mockClear();
  return render(
    <ToastProvider>
      <AuthFlashToast />
    </ToastProvider>,
  );
}

describe("AuthFlashToast", () => {
  // Achado de auditoria: logout e login terminavam sem nenhuma
  // confirmação visual — este componente é o que fecha essa lacuna.
  it("logout=success mostra o toast de saída e limpa o parâmetro da URL", () => {
    renderWithParams("logout=success");

    expect(screen.getByText("Você saiu com segurança")).toBeInTheDocument();
    expect(replace).toHaveBeenCalledWith("/", { scroll: false });
  });

  it("welcome=1 mostra o toast de boas-vindas e limpa o parâmetro da URL", () => {
    renderWithParams("welcome=1");

    expect(screen.getByText("Bem-vindo(a) de volta!")).toBeInTheDocument();
    expect(replace).toHaveBeenCalledWith("/", { scroll: false });
  });

  it("preserva outros parâmetros da URL ao limpar só o próprio", () => {
    renderWithParams("logout=success&foo=bar");

    expect(replace).toHaveBeenCalledWith("/?foo=bar", { scroll: false });
  });

  it("sem parâmetro reconhecido, não mostra toast nem mexe na URL", () => {
    renderWithParams("");

    expect(screen.queryByText("Você saiu com segurança")).not.toBeInTheDocument();
    expect(screen.queryByText("Bem-vindo(a) de volta!")).not.toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });
});
