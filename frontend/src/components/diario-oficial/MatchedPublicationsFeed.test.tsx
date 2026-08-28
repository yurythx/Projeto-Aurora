import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SWRConfig } from "swr";

import { ToastProvider } from "@/components/notifications/ToastProvider";

import { MatchedPublicationsFeed } from "./MatchedPublicationsFeed";

function mockFetchByPath(routes: Record<string, { status: number; body: unknown }>) {
  const fn = vi.fn().mockImplementation((url: string) => {
    const match = Object.entries(routes).find(([path]) => url.includes(path));
    if (!match) {
      return Promise.resolve({ ok: true, status: 200, json: async () => ({ data: [], error: null }) });
    }
    const [, { status, body }] = match;
    return Promise.resolve({ ok: status >= 200 && status < 300, status, json: async () => body });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

function renderFeed() {
  return render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <ToastProvider>
        <MatchedPublicationsFeed />
      </ToastProvider>
    </SWRConfig>,
  );
}

describe("MatchedPublicationsFeed", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("exibe publicações do Diário Oficial de Rondonópolis", async () => {
    mockFetchByPath({
      "monitored-terms": { status: 200, body: { data: [{ id: "t1", label: "Nomeações", active: true, created_at: "2026-08-01T00:00:00Z" }], error: null } },
      publications: {
        status: 200,
        body: {
          data: [
            {
              id: "p1",
              tribunal: "Secretaria Municipal de Administração",
              orgao: "Gabinete do Prefeito",
              tipo_comunicacao: "Extrato de Portaria",
              texto: "PORTARIA Nº 42.100 - Nomeação de servidor",
              process_number: "Portaria 42.100",
              process_number_masked: "Edição Nº 6.263 · 26/08/2026",
              availability_date: "2026-08-26",
              link: "https://www.rondonopolis.mt.gov.br/diario-oficial/",
              monitored_term_id: "t1",
              monitored_term_label: "Nomeações",
              matched_at: "2026-08-26T12:00:00Z",
            },
          ],
          error: null,
          meta: { page: 1, page_size: 20, total_items: 1, total_pages: 1 },
        },
      },
    });
    renderFeed();

    expect(await screen.findByText("Secretaria Municipal de Administração")).toBeInTheDocument();
    expect(screen.getByText("PORTARIA Nº 42.100 - Nomeação de servidor")).toBeInTheDocument();
    expect(screen.getAllByText("Nomeações")[0]).toBeInTheDocument();
    expect(screen.getByText("Abrir Diário Oficial (PDF) →")).toBeInTheDocument();
  });
});
