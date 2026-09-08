"use client";

import { useCallback, useEffect } from "react";

import { useNotifications } from "@/hooks/useNotifications";
import { useNotificationHistory } from "@/components/notifications/NotificationHistoryProvider";
import { useToast } from "@/components/notifications/ToastProvider";
import type { ToastTone } from "@/components/ui/Toast";
import {
  integrationStatusPayloadSchema,
  jobEventPayloadSchema,
  type EventEnvelope,
} from "@/lib/validation/schemas";
import type { ConnectionState } from "@/lib/websocket/client";

const eventCopy: Partial<Record<string, { title: string; tone: "success" | "danger" | "info" }>> = {
  "integration.test.completed": { title: "Teste de integração concluído", tone: "success" },
  "job.completed": { title: "Job de sistema concluído com sucesso", tone: "success" },
  "job.failed": { title: "Job de sistema falhou", tone: "danger" },
  "notification.created": { title: "Nova notificação do sistema", tone: "info" },
};

/** Aviso só em dev quando o payload de um evento não bate com o schema —
 * mismatch de contrato entre backend e frontend. Em produção fica silencioso
 * (o evento é ignorado sem quebrar a UI). Fora do componente: não depende de
 * nada dele. */
function logParseFailure(eventType: string, error: unknown) {
  if (process.env.NODE_ENV !== "production") {
    console.warn(`[NotificationCenter] Payload inválido para o evento '${eventType}':`, error);
  }
}

/** Montado uma única vez no layout do dashboard — não renderiza nada além
 * da pilha de toasts (via ToastProvider); traduz os eventos de WebSocket
 * já validados em toasts (§43). */
export function NotificationCenter({
  onConnectionStateChange,
}: {
  onConnectionStateChange?: (state: ConnectionState) => void;
}) {
  const { showToast } = useToast();
  const { push: pushHistory } = useNotificationHistory();

  const handleEvent = useCallback(
    (event: EventEnvelope) => {
      switch (event.type) {
        case "integration.status.changed": {
          const result = integrationStatusPayloadSchema.safeParse(event.payload);
          if (!result.success) {
            logParseFailure(event.type, result.error);
            return;
          }
          const tone: ToastTone = result.data.status === "online" ? "success" : "danger";
          const notification = {
            title: `Integração ${result.data.key} agora está ${result.data.status}`,
            tone,
          };
          showToast(notification);
          pushHistory(notification);
          return;
        }

        default: {
          const copy = eventCopy[event.type];
          if (!copy) return; // tipo de evento não reconhecido — ignora

          const job = jobEventPayloadSchema.safeParse(event.payload);
          const notification = {
            title: copy.title,
            description: job.success ? `Job ${job.data.job_id.slice(0, 8)}` : undefined,
            tone: copy.tone,
          };
          showToast(notification);
          pushHistory(notification);
        }
      }
    },
    [showToast, pushHistory],
  );

  const state = useNotifications(handleEvent);

  useEffect(() => {
    onConnectionStateChange?.(state);
  }, [state, onConnectionStateChange]);

  return null;
}
