"use client";

import { useCallback, useEffect } from "react";

import { useNotifications } from "@/hooks/useNotifications";
import { useNotificationHistory } from "@/components/notifications/NotificationHistoryProvider";
import { useToast } from "@/components/notifications/ToastProvider";
import type { ToastTone } from "@/components/ui/Toast";
import {
  integrationStatusPayloadSchema,
  jobEventPayloadSchema,
  demandEtapaChangedPayloadSchema,
  scanCompletedPayloadSchema,
  contratoDiarioRefLinkedPayloadSchema,
  contratoFiscalAlertPayloadSchema,
  type EventEnvelope,
} from "@/lib/validation/schemas";
import { mutate } from "swr";
import type { ConnectionState } from "@/lib/websocket/client";

const eventCopy: Partial<Record<string, { title: string; tone: "success" | "danger" | "info" }>> = {
  "diario_oficial.job.completed": { title: "Verificação do Diário Oficial concluída", tone: "success" },
  "diario_oficial.job.failed": { title: "Verificação do Diário Oficial falhou", tone: "danger" },
  "integration.test.completed": { title: "Teste de integração concluído", tone: "success" },
  "notification.created": { title: "Nova notificação", tone: "info" },
};

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
      if (event.type === "integration.status.changed") {
        const result = integrationStatusPayloadSchema.safeParse(event.payload);
        if (result.success) {
          const tone: ToastTone = result.data.status === "online" ? "success" : "danger";
          const notification = {
            title: `Integração ${result.data.key} agora está ${result.data.status}`,
            tone,
          };
          showToast(notification);
          pushHistory(notification);
        }
        return;
      }
      if (event.type === "demand.etapa_changed") {
        const result = demandEtapaChangedPayloadSchema.safeParse(event.payload);
        if (result.success) {
          const notification = {
            title: `Demanda de Contrato movida para Etapa ${result.data.new_etapa}`,
            tone: "info" as ToastTone,
          };
          showToast(notification);
          pushHistory(notification);
          // Força o KanbanBoard (que usa useSWR("/api/v1/demands/kanban")) a buscar os dados novos 
          mutate("/api/v1/demands/kanban");
        }
        return;
      }
      if (event.type === "contrato.diario_ref.linked") {
        const result = contratoDiarioRefLinkedPayloadSchema.safeParse(event.payload);
        if (result.success) {
          const n = result.data.refs_vinculadas;
          const notification = {
            title: `Contrato ${result.data.contrato_numero}: ${n} publicação${n > 1 ? "ões" : ""} do Diário vinculada${n > 1 ? "s" : ""}`,
            tone: "info" as ToastTone,
          };
          showToast(notification);
          pushHistory(notification);
          mutate("/api/v1/contratos/kanban");
        }
        return;
      }
      if (event.type === "contrato.fiscal_alert") {
        const result = contratoFiscalAlertPayloadSchema.safeParse(event.payload);
        if (result.success) {
          const notification = {
            title: `Alerta de fiscalização — contrato ${result.data.contrato_numero}`,
            description: result.data.message,
            tone: "danger" as ToastTone,
          };
          showToast(notification);
          pushHistory(notification);
        }
        return;
      }
      if (event.type === "scanning.scan.completed") {
        const result = scanCompletedPayloadSchema.safeParse(event.payload);
        if (result.success) {
          const { scanners, findings_count, critical_count, high_count } = result.data;
          let description = "Nenhum achado.";
          let tone: ToastTone = "success";

          if (findings_count > 0) {
            if (critical_count > 0) {
              description = `${findings_count} achado(s), ${critical_count} crítico(s)! — veja em Segurança.`;
              tone = "danger";
            } else if (high_count > 0) {
              description = `${findings_count} achado(s), ${high_count} alto(s) — veja em Segurança.`;
              tone = "danger";
            } else {
              description = `${findings_count} achado(s) — veja em Segurança.`;
              tone = "info";
            }
          }

          const notification = {
            title: `Scan concluído (${scanners.join(", ")})`,
            description,
            tone,
          };
          showToast(notification);
          pushHistory(notification);
        }
        return;
      }

      const copy = eventCopy[event.type];
      if (!copy) return; // tipo de evento não reconhecido — ignora em vez de adivinhar

      const job = jobEventPayloadSchema.safeParse(event.payload);
      const notification = {
        title: copy.title,
        description: job.success ? `Job ${job.data.job_id.slice(0, 8)}` : undefined,
        tone: copy.tone,
      };
      showToast(notification);
      pushHistory(notification);
    },
    [showToast, pushHistory],
  );

  const state = useNotifications(handleEvent);

  useEffect(() => {
    onConnectionStateChange?.(state);
  }, [state, onConnectionStateChange]);

  return null;
}
