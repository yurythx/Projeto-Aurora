import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const signIn = vi.fn();
vi.mock("next-auth/react", () => ({ signIn: (...a: unknown[]) => signIn(...a) }));

const push = vi.fn();
let searchParams = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
  useSearchParams: () => searchParams,
}));

import { LoginCard } from "./LoginCard";

describe("LoginCard", () => {
  beforeEach(() => {
    searchParams = new URLSearchParams();
    signIn.mockReset();
    push.mockReset();
  });
  afterEach(() => vi.clearAllMocks());

  it("alterna a visibilidade da senha", async () => {
    const user = userEvent.setup();
    render(<LoginCard />);
    const senha = screen.getByLabelText("Senha") as HTMLInputElement;
    expect(senha.type).toBe("password");

    await user.click(screen.getByRole("button", { name: "Mostrar senha" }));
    expect(senha.type).toBe("text");

    await user.click(screen.getByRole("button", { name: "Ocultar senha" }));
    expect(senha.type).toBe("password");
  });

  it("credenciais inválidas: mostra mensagem genérica e não navega", async () => {
    signIn.mockResolvedValue({ error: "CredentialsSignin" });
    const user = userEvent.setup();
    render(<LoginCard />);

    await user.type(screen.getByLabelText("Usuário"), "admin");
    await user.type(screen.getByLabelText("Senha"), "errada");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    expect(await screen.findByText("Usuário ou senha inválidos.")).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
  });

  it("sucesso: navega para o callbackUrl", async () => {
    searchParams = new URLSearchParams("callbackUrl=/dashboard");
    signIn.mockResolvedValue({ ok: true, error: null });
    const user = userEvent.setup();
    render(<LoginCard />);

    await user.type(screen.getByLabelText("Usuário"), "admin");
    await user.type(screen.getByLabelText("Senha"), "Admin123!");
    await user.click(screen.getByRole("button", { name: "Entrar" }));

    await vi.waitFor(() => expect(push).toHaveBeenCalledWith("/dashboard"));
    expect(signIn).toHaveBeenCalledWith(
      "local",
      expect.objectContaining({ username: "admin", password: "Admin123!", redirect: false }),
    );
  });

  it("o botão de SSO dispara signIn('keycloak')", async () => {
    const user = userEvent.setup();
    render(<LoginCard />);
    await user.click(screen.getByRole("button", { name: "Entrar com SSO corporativo" }));
    expect(signIn).toHaveBeenCalledWith("keycloak", expect.objectContaining({ callbackUrl: "/dashboard" }));
  });

  it("erro de OAuth vindo pela URL mostra o alerta", () => {
    searchParams = new URLSearchParams("error=OAuthCallback");
    render(<LoginCard />);
    expect(screen.getByRole("alert")).toHaveTextContent("Falha ao entrar");
  });
});
