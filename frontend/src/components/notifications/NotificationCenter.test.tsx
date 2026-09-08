import { act, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { EventEnvelope } from "@/lib/validation/schemas";

// useNotifications é quem de fato fala com o WebSocket — mockado aqui
// pra capturar o handleEvent que NotificationCenter registra, e chamá-lo
// diretamente com eventos sintéticos, sem precisar de um servidor
// WebSocket de verdade rodando no teste.
const { useNotifications } = vi.hoisted(() => ({ useNotifications: vi.fn() }));
vi.mock("@/hooks/useNotifications", () => ({ useNotifications }));

import { NotificationHistoryProvider, useNotificationHistory } from "./NotificationHistoryProvider";
import { NotificationCenter } from "./NotificationCenter";
import { ToastProvider } from "./ToastProvider";

function envelope(type: string, payload: unknown): EventEnvelope {
  return {
    id: "evt-1",
    type,
    version: 1,
    source: "backend-worker",
    occurred_at: "2026-08-25T12:00:00Z",
    correlation_id: "corr-1",
    payload,
  };
}

// Exibe a contagem de itens no histórico, pra provar que pushHistory foi
// chamado (não só showToast) — mesmo princípio do Probe já usado em
// NotificationHistoryProvider.test.tsx.
function HistoryCount() {
  const { items } = useNotificationHistory();
  return <p data-testid="history-count">{items.length}</p>;
}

let capturedHandler: ((event: EventEnvelope) => void) | undefined;

function renderCenter() {
  useNotifications.mockImplementation((onEvent: (event: EventEnvelope) => void) => {
    capturedHandler = onEvent;
    return "open";
  });
  return render(
    <ToastProvider>
      <NotificationHistoryProvider>
        <HistoryCount />
        <NotificationCenter />
      </NotificationHistoryProvider>
    </ToastProvider>,
  );
}

describe("NotificationCenter", () => {
  it("integration.status.changed vira toast + histórico com o tom certo por status", () => {
    renderCenter();
    act(() =>
      capturedHandler!(
        envelope("integration.status.changed", { key: "example-service", status: "online" }),
      ),
    );

    expect(screen.getByText("Integração example-service agora está online")).toBeInTheDocument();
    expect(screen.getByTestId("history-count")).toHaveTextContent("1");
  });

  it("eventos de job (job.completed) usam o schema genérico de job_id", () => {
    renderCenter();
    act(() =>
      capturedHandler!(envelope("job.completed", { job_id: "12345678-abcd-def0" })),
    );

    expect(screen.getByText("Job de sistema concluído com sucesso")).toBeInTheDocument();
    expect(screen.getByText("Job 12345678")).toBeInTheDocument();
  });

  it("um tipo de evento desconhecido é ignorado, sem toast nem histórico", () => {
    renderCenter();
    act(() => capturedHandler!(envelope("something.nobody.emits", { whatever: true })));

    expect(screen.getByTestId("history-count")).toHaveTextContent("0");
  });

  it("um payload que não bate com o schema esperado é descartado, sem quebrar nem gerar toast", () => {
    renderCenter();
    act(() =>
      capturedHandler!(
        envelope("integration.status.changed", { key: "example-service", status: "esquisito" }),
      ),
    );

    expect(screen.getByTestId("history-count")).toHaveTextContent("0");
  });

  it("repassa o ConnectionState de useNotifications pro callback onConnectionStateChange", () => {
    useNotifications.mockImplementation(() => "closed");
    const onConnectionStateChange = vi.fn();
    render(
      <ToastProvider>
        <NotificationHistoryProvider>
          <NotificationCenter onConnectionStateChange={onConnectionStateChange} />
        </NotificationHistoryProvider>
      </ToastProvider>,
    );

    expect(onConnectionStateChange).toHaveBeenCalledWith("closed");
  });
});
