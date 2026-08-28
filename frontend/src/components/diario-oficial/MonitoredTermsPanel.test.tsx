import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SWRConfig } from "swr";

import { ToastProvider } from "@/components/notifications/ToastProvider";

import { MonitoredTermsPanel } from "./MonitoredTermsPanel";

function mockFetchOnce(status: number, body: unknown) {
  const fn = vi.fn().mockResolvedValue({ ok: status >= 200 && status < 300, status, json: async () => body });
  vi.stubGlobal("fetch", fn);
  return fn;
}

function mockFetchSequence(...responses: { status: number; body: unknown }[]) {
  const fetchMock = vi.fn();
  for (const { status, body } of responses) {
    fetchMock.mockResolvedValueOnce({ ok: status >= 200 && status < 300, status, json: async () => body });
  }
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderPanel() {
  return render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <ToastProvider>
        <MonitoredTermsPanel />
      </ToastProvider>
    </SWRConfig>,
  );
}

describe("MonitoredTermsPanel", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sem termo nenhum, mostra o estado vazio", async () => {
    mockFetchOnce(200, { data: [], error: null });
    renderPanel();

    expect(await screen.findByText("Nenhum termo monitorado ainda")).toBeInTheDocument();
  });

  it("lista um termo cadastrado com o selo Ativo", async () => {
    mockFetchOnce(200, {
      data: [
        {
          id: "t1",
          label: "Lucas",
          free_text: "Lucas",
          active: true,
          last_synced_at: "2026-08-26T12:00:00Z",
          created_at: "2026-08-01T00:00:00Z",
        },
      ],
      error: null,
    });
    renderPanel();

    expect(await screen.findByText("Lucas")).toBeInTheDocument();
    expect(screen.getByText("Ativo")).toBeInTheDocument();
  });

  it("cadastra um termo por nome e limpa o formulário", async () => {
    const user = userEvent.setup();
    const fetchMock = mockFetchSequence(
      { status: 200, body: { data: [], error: null } }, // carga inicial (GET)
      { status: 201, body: { data: { id: "t2", label: "Lucas", active: true, created_at: "2026-08-26T00:00:00Z" }, error: null } }, // POST
      { status: 200, body: { data: [{ id: "t2", label: "Lucas", active: true, created_at: "2026-08-26T00:00:00Z" }], error: null } }, // revalidação
    );
    renderPanel();
    await screen.findByText("Nenhum termo monitorado ainda");

    await user.type(screen.getByLabelText("Nome de Servidor, Empresa ou Palavra-chave Municipal"), "Lucas");
    await user.click(screen.getByRole("button", { name: "Adicionar Termo" }));

    expect(await screen.findByText("Termo cadastrado com sucesso")).toBeInTheDocument();
    await waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2));

    const [, postInit] = fetchMock.mock.calls[1] ?? [];
    const sentBody = JSON.parse((postInit as RequestInit).body as string);
    expect(sentBody).toMatchObject({ label: "Lucas", free_text: "Lucas" });
  });

  it("remover um termo chama DELETE com o id certo", async () => {
    const user = userEvent.setup();
    const fetchMock = mockFetchSequence(
      { status: 200, body: { data: [{ id: "t3", label: "termo a remover", active: true, created_at: "2026-08-26T00:00:00Z" }], error: null } },
      { status: 200, body: { data: {}, error: null } }, // DELETE
      { status: 200, body: { data: [], error: null } }, // revalidação
    );
    renderPanel();
    await screen.findByText("termo a remover");

    await user.click(screen.getByRole("button", { name: /Remover/i }));

    expect(await screen.findByText("Termo removido")).toBeInTheDocument();
    const [deleteUrl, deleteInit] = fetchMock.mock.calls[1] ?? [];
    expect(deleteUrl).toContain("diario-oficial/monitored-terms/t3");
    expect((deleteInit as RequestInit).method).toBe("DELETE");
  });
});
