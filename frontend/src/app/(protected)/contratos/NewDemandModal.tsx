"use client";

import { useEffect, useState } from "react";
import { X, Calendar, FileText, Building2 } from "lucide-react";
import { createDemand } from "@/lib/api/demands";
import { fetchContratos } from "@/lib/api/contratos";
import type { Contrato } from "@/types/api";
import { Button } from "@/components/ui/Button";
import { ModalShell } from "@/components/ui/ModalShell";
import { useToast } from "@/components/notifications/ToastProvider";

interface Props {
  onClose: () => void;
  onCreated: () => void;
}

export function NewDemandModal({ onClose, onCreated }: Props) {
  const [contratos, setContratos] = useState<Contrato[]>([]);
  const [loadingContratos, setLoadingContratos] = useState(true);
  const [selectedContratoId, setSelectedContratoId] = useState("");
  const [anoMes, setAnoMes] = useState(new Date().toISOString().slice(0, 7)); // e.g. "2026-08"
  const [observacoes, setObservacoes] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const { showToast } = useToast();

  useEffect(() => {
    async function loadContratos() {
      try {
        // O endpoint de listagem devolve um envelope paginado
        // ({ data, total, page, pages }); fetchContratos já o desembrulha.
        // Chamar apiClient direto aqui deixava `contratos` como objeto e
        // quebrava o .map do <select> ("contratos.map is not a function").
        const { data } = await fetchContratos(1, 200);
        const list = Array.isArray(data) ? data : [];
        setContratos(list);
        if (list.length > 0 && list[0]) {
          setSelectedContratoId(list[0].id);
        }
      } catch {
        showToast({
          title: "Erro ao carregar contratos",
          description: "Não foi possível carregar a lista de contratos cadastrados.",
          tone: "danger",
        });
      } finally {
        setLoadingContratos(false);
      }
    }
    loadContratos();
  }, [showToast]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedContratoId) {
      showToast({
        title: "Campo obrigatório",
        description: "Selecione um contrato para iniciar a demanda.",
        tone: "info",
      });
      return;
    }
    if (!anoMes) {
      showToast({
        title: "Campo obrigatório",
        description: "Informe o mês/ano de competência (AAAA-MM).",
        tone: "info",
      });
      return;
    }

    setSubmitting(true);
    try {
      await createDemand(selectedContratoId, anoMes, observacoes);
      showToast({
        title: "Demanda Criada com Sucesso",
        description: `Iniciado o fluxo de liquidação para o mês ${anoMes}.`,
        tone: "success",
      });
      onCreated();
      onClose();
    } catch (err) {
      showToast({
        title: "Falha ao criar demanda",
        description:
          err instanceof Error ? err.message : "Ocorreu um erro ao registrar a demanda mensal.",
        tone: "danger",
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ModalShell open onClose={onClose} size="lg" labelledBy="new-demand-title">
      {/* Header */}
      <div className="flex shrink-0 items-center justify-between border-b border-surface-border px-6 py-4">
        <div>
          <h2 id="new-demand-title" className="text-lg font-semibold">
            Nova Demanda de Liquidação
          </h2>
          <p className="text-xs text-muted">
            Inicia o fluxo de 6 etapas da IN SCL 01/2019 para o contrato.
          </p>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Fechar modal"
          className="rounded-lg p-1.5 text-muted hover:bg-surface-hover hover:text-foreground"
        >
          <X size={18} />
        </button>
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-6">
          {/* Seleção do Contrato */}
          <div>
            <label className="block text-xs font-medium text-muted uppercase tracking-wider mb-1.5">
              Contrato de Origem *
            </label>
            {loadingContratos ? (
              <div className="h-10 w-full animate-pulse rounded-md bg-surface-hover/50" />
            ) : (
              <div className="relative">
                <Building2 className="absolute left-3 top-3 h-4 w-4 text-muted pointer-events-none" />
                <select
                  value={selectedContratoId}
                  onChange={(e) => setSelectedContratoId(e.target.value)}
                  className="w-full rounded-md border border-surface-border bg-surface pl-10 pr-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  required
                >
                  {contratos.length === 0 ? (
                    <option value="">Nenhum contrato cadastrado</option>
                  ) : (
                    contratos.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.numero} — {c.contratado} ({c.objeto?.substring(0, 40)}…)
                      </option>
                    ))
                  )}
                </select>
              </div>
            )}
          </div>

          {/* Competência (Mês/Ano) */}
          <div>
            <label className="block text-xs font-medium text-muted uppercase tracking-wider mb-1.5">
              Competência (Ano-Mês) *
            </label>
            <div className="relative">
              <Calendar className="absolute left-3 top-3 h-4 w-4 text-muted pointer-events-none" />
              <input
                type="month"
                value={anoMes}
                onChange={(e) => setAnoMes(e.target.value)}
                className="w-full rounded-md border border-surface-border bg-surface pl-10 pr-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                required
              />
            </div>
          </div>

          {/* Observações */}
          <div>
            <label className="block text-xs font-medium text-muted uppercase tracking-wider mb-1.5">
              Observações Iniciais
            </label>
            <div className="relative">
              <FileText className="absolute left-3 top-3 h-4 w-4 text-muted pointer-events-none" />
              <textarea
                rows={3}
                value={observacoes}
                onChange={(e) => setObservacoes(e.target.value)}
                placeholder="Ex: Medição referente ao período de 01 a 31..."
                className="w-full rounded-md border border-surface-border bg-surface pl-10 pr-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>
          </div>

          {/* Actions — rodapé fixo */}
          <div className="flex shrink-0 items-center justify-end gap-3 border-t border-surface-border px-6 py-4">
            <Button type="button" variant="ghost" onClick={onClose} disabled={submitting}>
              Cancelar
            </Button>
            <Button type="submit" disabled={submitting || loadingContratos || contratos.length === 0}>
              {submitting ? "Criando..." : "Criar Demanda"}
            </Button>
          </div>
        </form>
    </ModalShell>
  );
}
