import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SWRConfig } from "swr";

import { ToastProvider } from "@/components/notifications/ToastProvider";
import type { PublicContract } from "@/types/api";

import { RondonopolisContractsFeed } from "./RondonopolisContractsFeed";

const contract: PublicContract = {
  id: "c1",
  contract_number: "Contrato nº 140/2026",
  contract_type: "Tecnologia da Informação",
  status: "NOVO",
  value: "R$ 950.000,00",
  contractor: "TechGov Soluções LTDA",
  contractor_cnpj: "03.492.110/0001-99",
  fiscal_nome: "MARIA APARECIDA SOUZA",
  fiscal_cpf: "021.946.881-88",
  fiscal_matricula: "MAT-2025-081",
  suplente_nome: "",
  suplente_cpf: "",
  suplente_matricula: "",
  portaria_number: "PORT-491/2026",
  edition_number: "6263",
  publication_date: "2026-08-20T00:00:00Z",
  nomeacao_date: "2026-08-20T00:00:00Z",
  object: "Serviços continuados de modernização tecnológica.",
  doc_url: "https://example.gov/edicao.pdf",
};

function mockFetch(body: PublicContract[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data: body, error: null }) }),
  );
}

function renderFeed() {
  return render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <ToastProvider>
        <RondonopolisContractsFeed />
      </ToastProvider>
    </SWRConfig>,
  );
}

describe("RondonopolisContractsFeed", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("enquanto carrega, mostra o texto de carregando", () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => {})));
    renderFeed();
    expect(screen.getByText(/Carregando contratos públicos/)).toBeInTheDocument();
  });

  it("renderiza um contrato retornado pela API", async () => {
    mockFetch([contract]);
    renderFeed();
    expect(await screen.findByText("Contrato nº 140/2026")).toBeInTheDocument();
    expect(screen.getByText("TechGov Soluções LTDA")).toBeInTheDocument();
  });

  it("lista vazia mostra o estado vazio", async () => {
    mockFetch([]);
    renderFeed();
    expect(await screen.findByText("Não achamos nenhuma referência")).toBeInTheDocument();
  });

  it("abre o modal de detalhes ao clicar em 'Ver Detalhes' e fecha no backdrop", async () => {
    mockFetch([contract]);
    const user = userEvent.setup();
    renderFeed();
    await screen.findByText("Contrato nº 140/2026");

    await user.click(screen.getByRole("button", { name: /Ver Detalhes do Contrato/ }));
    expect(
      await screen.findByText("Extrato de Auditoria de Contrato Público"),
    ).toBeInTheDocument();
    // o nome do fiscal aparece no corpo do modal
    expect(screen.getAllByText("MARIA APARECIDA SOUZA").length).toBeGreaterThan(0);

    // clique no <dialog> (backdrop) fecha
    await user.click(document.querySelector("dialog")!);
    await vi.waitFor(() =>
      expect(screen.queryByText("Extrato de Auditoria de Contrato Público")).not.toBeInTheDocument(),
    );
  });

  it("filtra a lista pela busca textual", async () => {
    mockFetch([
      contract,
      { ...contract, id: "c2", contract_number: "Contrato nº 999/2026", contractor: "Outra Empresa SA" },
    ]);
    const user = userEvent.setup();
    renderFeed();
    await screen.findByText("Contrato nº 140/2026");

    await user.type(screen.getByPlaceholderText(/Buscar/i), "999");
    await vi.waitFor(() => {
      expect(screen.queryByText("Contrato nº 140/2026")).not.toBeInTheDocument();
      expect(screen.getByText("Contrato nº 999/2026")).toBeInTheDocument();
    });
  });
});
