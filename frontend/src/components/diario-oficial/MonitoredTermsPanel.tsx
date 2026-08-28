"use client";

import { useState, type FormEvent } from "react";

import { useToast } from "@/components/notifications/ToastProvider";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { ApiError, useApiQuery } from "@/lib/api/swr";
import type { MonitoredTerm } from "@/types/api";

// MonitoredTermsPanel — Cadastro e acompanhamento de termos e nomes monitorados
// no Diário Oficial de Rondonópolis (DIORONDON-E).
export function MonitoredTermsPanel() {
  const { data: terms, error, isLoading, mutate } = useApiQuery<MonitoredTerm[]>("v1/diario-oficial/monitored-terms");
  const { showToast } = useToast();

  const [termName, setTermName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!termName.trim()) return;

    setSubmitting(true);
    try {
      await apiClient.post("v1/diario-oficial/monitored-terms", {
        label: termName.trim(),
        free_text: termName.trim(),
      });
      showToast({ title: "Termo cadastrado com sucesso", description: termName.trim(), tone: "info" });
      setTermName("");
      void mutate();
    } catch (err) {
      showToast({
        title: "Não foi possível cadastrar o termo",
        description: err instanceof ApiError ? err.message : "Erro inesperado",
        tone: "danger",
      });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(term: MonitoredTerm) {
    setDeletingID(term.id);
    try {
      await apiClient.delete(`v1/diario-oficial/monitored-terms/${term.id}`);
      showToast({ title: "Termo removido", description: term.label, tone: "info" });
      void mutate();
    } catch (err) {
      showToast({
        title: "Não foi possível remover o termo",
        description: err instanceof ApiError ? err.message : "Erro inesperado",
        tone: "danger",
      });
    } finally {
      setDeletingID(null);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4 rounded-lg border border-surface-border bg-surface p-4">
        <div className="flex flex-col sm:flex-row items-end gap-3">
          <div className="flex-1 w-full">
            <Input
              label="Nome de Servidor, Empresa ou Palavra-chave Municipal"
              name="termName"
              placeholder="ex.: Vanete, Marileide, Soluções em Engenharia, Secretaria de Obras"
              value={termName}
              onChange={(e) => setTermName(e.target.value)}
              required
            />
          </div>
          <Button type="submit" loading={submitting} className="w-full sm:w-auto shrink-0">
            Adicionar Termo
          </Button>
        </div>
      </form>

      {error ? (
        <p className="text-sm text-danger">
          {error instanceof ApiError ? error.message : "Falha ao carregar termos monitorados"}
        </p>
      ) : isLoading && !terms ? (
        <p className="text-sm text-muted">Carregando termos monitorados…</p>
      ) : !terms || terms.length === 0 ? (
        <EmptyState
          title="Nenhum termo monitorado ainda"
          description="Cadastre um nome de servidor, empresa ou palavra-chave acima — o sistema monitora contra o Diário Oficial de Rondonópolis."
        />
      ) : (
        <ul className="flex flex-col divide-y divide-surface-border rounded-lg border border-surface-border bg-surface">
          {terms.map((term) => (
            <li key={term.id} className="flex items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <div className="truncate font-medium text-foreground">{term.label}</div>
                <div className="truncate text-xs text-muted">
                  {term.free_text || term.label}
                  {" · "}
                  {term.last_synced_at
                    ? `última sincronização ${new Date(term.last_synced_at).toLocaleString("pt-BR")}`
                    : "sincronizado automaticamente"}
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Badge tone={term.active ? "success" : "neutral"}>
                  {term.active ? "Ativo" : "Inativo"}
                </Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  loading={deletingID === term.id}
                  onClick={() => void handleDelete(term)}
                  aria-label={`Remover termo ${term.label}`}
                >
                  Remover
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
