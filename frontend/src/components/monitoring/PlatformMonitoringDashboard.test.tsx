import { render, screen } from "@testing-library/react";
import { SWRConfig } from "swr";
import { afterEach, describe, expect, it, vi } from "vitest";

import { PlatformMonitoringDashboard } from "./PlatformMonitoringDashboard";

// Mocka fetch("/api/health") como a própria rota (app/api/health/route.ts)
// responde: {data: {status, timestamp, services}, error}. checks é
// {postgres: "ok"|"unavailable", ...} — o mesmo formato bruto do /ready
// do backend, já reformatado como a rota faz.
function mockHealth(status: number, checks: Record<string, string>) {
  const services = Object.fromEntries(
    Object.entries(checks).map(([name, s]) => [name, { status: s }]),
  );
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      json: async () => ({
        data: { status: status === 200 ? "ok" : "degraded", timestamp: new Date().toISOString(), services },
        error: null,
      }),
    }),
  );
}

// SWR mantém um cache GLOBAL por chave ("/api/health") entre renders — sem
// isto, o resultado de um teste anterior (ex.: postgres/rabbitmq "ok")
// continuava visível como dado "stale" no início do próximo teste, antes
// do novo mock de fetch sequer resolver, produzindo falsos positivos.
// `provider: () => new Map()` dá um cache novo e isolado a cada render.
function renderDashboard() {
  return render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <PlatformMonitoringDashboard />
    </SWRConfig>,
  );
}

describe("PlatformMonitoringDashboard", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // Achado de auditoria corrigido: os quatro cartões de infraestrutura
  // mostravam "online" fixo no código-fonte, sempre — nunca refletiam o
  // /ready de verdade do backend. Este teste cobre a derivação honesta:
  // um serviço que o /ready reporta "unavailable" precisa aparecer como
  // tal na tela, não como "Online".
  it("reflete o status real do backend — rabbitmq indisponível aparece como Offline", async () => {
    mockHealth(503, { postgres: "ok", rabbitmq: "unavailable" });
    renderDashboard();

    expect(await screen.findByText("PostgreSQL 16 Engine")).toBeInTheDocument();

    const rabbitCard = await screen.findByText("RabbitMQ AMQP Broker");
    expect(rabbitCard.closest(".relative")!.textContent).toContain("Offline");

    const postgresCard = screen.getByText("PostgreSQL 16 Engine");
    expect(postgresCard.closest(".relative")!.textContent).toContain("Online");
  });

  it("MinIO nunca é reportado como 'Online' sem uma checagem real — aparece como Desconhecido", async () => {
    mockHealth(200, { postgres: "ok", rabbitmq: "ok" });
    renderDashboard();

    await screen.findByText("PostgreSQL 16 Engine");
    const minioCard = await screen.findByText("MinIO Object Storage (S3)");
    expect(minioCard.closest(".relative")!.textContent).toContain("Desconhecido");
  });

  it("não existe mais um botão \"Testar\" chamando um endpoint de integração inexistente", async () => {
    mockHealth(200, { postgres: "ok", rabbitmq: "ok" });
    renderDashboard();

    await screen.findByText("PostgreSQL 16 Engine");
    expect(screen.queryByRole("button", { name: /testar/i })).not.toBeInTheDocument();
  });

  it("backend inalcançável — nenhum serviço aparece como Online", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("network error")));
    renderDashboard();

    await screen.findByText("PostgreSQL 16 Engine");
    await screen.findAllByText("Offline");
    expect(screen.queryAllByText("Online")).toHaveLength(0);
  });
});
