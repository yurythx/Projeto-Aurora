"use client";

import { useMemo, useState, useEffect } from "react";
import {
  Bell,
  Search,
  Plus,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  ShieldCheck,
  Pause,
  Play,
  ExternalLink,
  Check,
} from "lucide-react";

import { useToast } from "@/components/notifications/ToastProvider";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { useApiQuery } from "@/lib/api/swr";
import type { HREvent, PublicContract } from "@/types/api";

export type TargetType = "CPF" | "MATRICULA" | "NOME" | "CNPJ" | "TERMO_GERAL";

export interface MonitoredTarget {
  id: string;
  type: TargetType;
  value: string;
  label: string;
  active: boolean;
  notifyOnNomeacao: boolean;
  notifyOnExoneracao: boolean;
  notifyOnRelotacao: boolean;
  notifyOnContrato: boolean;
  createdAt: string;
}

export interface AuditAlert {
  id: string;
  targetId: string;
  targetValue: string;
  targetLabel: string;
  targetType: TargetType;
  title: string;
  snippet: string;
  editionNumber?: string;
  publicationDate?: string;
  docUrl?: string;
  read: boolean;
  createdAt: string;
}

const INITIAL_TARGETS: MonitoredTarget[] = [
  {
    id: "target-1",
    type: "CPF",
    value: "021.946.881-88",
    label: "Servidor Yuri - Auditoria de Nomeação & Contratos",
    active: true,
    notifyOnNomeacao: true,
    notifyOnExoneracao: true,
    notifyOnRelotacao: true,
    notifyOnContrato: true,
    createdAt: new Date().toISOString(),
  },
  {
    id: "target-2",
    type: "MATRICULA",
    value: "MAT-2025-03",
    label: "Matrícula Gabinete do Prefeito",
    active: true,
    notifyOnNomeacao: true,
    notifyOnExoneracao: true,
    notifyOnRelotacao: true,
    notifyOnContrato: false,
    createdAt: new Date().toISOString(),
  },
  {
    id: "target-3",
    type: "CNPJ",
    value: "03.492.110/0001-99",
    label: "TechSoluções Rondonópolis Ltda - Fiscalização Contratual",
    active: true,
    notifyOnNomeacao: false,
    notifyOnExoneracao: false,
    notifyOnRelotacao: false,
    notifyOnContrato: true,
    createdAt: new Date().toISOString(),
  },
];

const LOCAL_STORAGE_KEY = "nix_monitored_targets_v1";

/** Normaliza textos para correspondência simples */
function normalizeText(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase();
}

export function AuditMonitoringCenter() {
  const [targets, setTargets] = useState<MonitoredTarget[]>(() => {
    if (typeof window !== "undefined") {
      const stored = localStorage.getItem(LOCAL_STORAGE_KEY);
      if (stored) {
        try {
          return JSON.parse(stored);
        } catch {
          // Fallback para alvos iniciais
        }
      }
    }
    return INITIAL_TARGETS;
  });

  // Salva no localStorage sempre que os alvos mudam
  useEffect(() => {
    if (typeof window !== "undefined") {
      localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify(targets));
    }
  }, [targets]);

  // Form State para adicionar novo alvo
  const [newType, setNewType] = useState<TargetType>("CPF");
  const [newValue, setNewValue] = useState("");
  const [newLabel, setNewLabel] = useState("");
  const [notifyOnNomeacao, setNotifyOnNomeacao] = useState(true);
  const [notifyOnExoneracao, setNotifyOnExoneracao] = useState(true);
  const [notifyOnRelotacao, setNotifyOnRelotacao] = useState(true);
  const [notifyOnContrato, setNotifyOnContrato] = useState(true);

  // Filtro de Alertas
  const [alertFilter, setAlertFilter] = useState<"ALL" | "UNREAD">("ALL");
  const [searchFilter, setSearchFilter] = useState("");

  const { showToast } = useToast();

  // Consultas de dados do backend para alimentar alertas reais
  const { data: hrEvents } = useApiQuery<HREvent[]>("v1/diario-oficial/rondonopolis/hr-events");
  const { data: contracts } = useApiQuery<PublicContract[]>("v1/diario-oficial/rondonopolis/contracts");

  const activeHrEvents = useMemo(() => {
    return Array.isArray(hrEvents) ? hrEvents : [];
  }, [hrEvents]);

  const activeContracts = useMemo(() => {
    return Array.isArray(contracts) ? contracts : [];
  }, [contracts]);

  // Geração dinâmica de alertas cruzando Alvos Salvos com o Banco/API
  const { alerts, alertCountByTarget } = useMemo(() => {
    const generatedAlerts: AuditAlert[] = [];
    const countMap: Record<string, number> = {};

    targets.forEach((t) => {
      if (!t.active) return;

      const targetDigits = t.value.replace(/\D/g, "");
      const normalizedVal = normalizeText(t.value);

      // Checa Atos de Pessoal
      if (activeHrEvents) {
        activeHrEvents.forEach((ev) => {
          const cpfDigits = (ev.servidor_cpf || "").replace(/\D/g, "");
          const matDigits = (ev.servidor_matricula || "").replace(/\D/g, "");
          const nameNorm = normalizeText(ev.servidor || "");

          let matches = false;

          if (t.type === "CPF" && targetDigits.length >= 3 && cpfDigits.includes(targetDigits)) {
            matches = true;
          } else if (t.type === "MATRICULA" && (targetDigits.length >= 3 && matDigits.includes(targetDigits) || normalizeText(ev.servidor_matricula || "").includes(normalizedVal))) {
            matches = true;
          } else if (t.type === "NOME" && nameNorm.includes(normalizedVal)) {
            matches = true;
          } else if (t.type === "TERMO_GERAL" && (nameNorm.includes(normalizedVal) || normalizeText(ev.context_snippet || "").includes(normalizedVal))) {
            matches = true;
          }

          if (matches) {
            const isExoneracao = ev.type === "EXONERACAO";
            const isNomeacao = ev.type === "NOMEACAO";
            const isRelotacao = ev.type === "MUDANCA_SETOR";

            const shouldNotify =
              (isExoneracao && t.notifyOnExoneracao) ||
              (isNomeacao && t.notifyOnNomeacao) ||
              (isRelotacao && t.notifyOnRelotacao);

            if (shouldNotify) {
              countMap[t.id] = (countMap[t.id] || 0) + 1;
              generatedAlerts.push({
                id: `alert-hr-${t.id}-${ev.portaria_number || Math.random()}`,
                targetId: t.id,
                targetValue: t.value,
                targetLabel: t.label,
                targetType: t.type,
                title: `${isExoneracao ? "🚨 Exoneração Publicada" : isNomeacao ? "✅ Nomeação / Contratação Publicada" : "🔄 Relotação Publicada"} — ${ev.servidor}`,
                snippet: `Servidor: ${ev.servidor} | CPF: ${ev.servidor_cpf || "N/A"} | Matrícula: ${ev.servidor_matricula || "N/A"}. Portaria Nº ${ev.portaria_number || "N/A"}. ${ev.cargo ? `Cargo: ${ev.cargo}.` : ""} "${ev.context_snippet}"`,
                editionNumber: ev.edition_number,
                publicationDate: ev.publication_date,
                docUrl: ev.doc_url,
                read: false,
                createdAt: ev.publication_date || new Date().toISOString(),
              });
            }
          }
        });
      }

      // Checa Contratos Públicos
      if (activeContracts && t.notifyOnContrato) {
        activeContracts.forEach((c) => {
          const fiscalCpfDigits = (c.fiscal_cpf || "").replace(/\D/g, "");
          const suplenteCpfDigits = (c.suplente_cpf || "").replace(/\D/g, "");
          const contractorCnpjDigits = (c.contractor_cnpj || "").replace(/\D/g, "");
          const fiscalMatDigits = (c.fiscal_matricula || "").replace(/\D/g, "");

          let matches = false;

          if (t.type === "CPF" && targetDigits.length >= 3 && (fiscalCpfDigits.includes(targetDigits) || suplenteCpfDigits.includes(targetDigits))) {
            matches = true;
          } else if (t.type === "MATRICULA" && targetDigits.length >= 3 && fiscalMatDigits.includes(targetDigits)) {
            matches = true;
          } else if (t.type === "CNPJ" && targetDigits.length >= 3 && contractorCnpjDigits.includes(targetDigits)) {
            matches = true;
          } else if (t.type === "NOME" && (normalizeText(c.fiscal_nome || "").includes(normalizedVal) || normalizeText(c.suplente_nome || "").includes(normalizedVal))) {
            matches = true;
          } else if (t.type === "TERMO_GERAL" && (normalizeText(c.object || "").includes(normalizedVal) || normalizeText(c.contractor || "").includes(normalizedVal))) {
            matches = true;
          }

          if (matches) {
            countMap[t.id] = (countMap[t.id] || 0) + 1;
            generatedAlerts.push({
              id: `alert-contract-${t.id}-${c.contract_number || Math.random()}`,
              targetId: t.id,
              targetValue: t.value,
              targetLabel: t.label,
              targetType: t.type,
              title: `📜 Fiscalização de Contrato Designada — ${c.contract_number}`,
              snippet: `Contrato ${c.contract_number} (${c.value}). Empresa: ${c.contractor} (CNPJ: ${c.contractor_cnpj}). Fiscal Titular: ${c.fiscal_nome} (CPF: ${c.fiscal_cpf}). Objeto: ${c.object}`,
              editionNumber: c.edition_number,
              publicationDate: c.publication_date,
              docUrl: c.doc_url,
              read: false,
              createdAt: c.publication_date || new Date().toISOString(),
            });
          }
        });
      }
    });

    return { alerts: generatedAlerts, alertCountByTarget: countMap };
  }, [targets, hrEvents, contracts]);

  // Alertas Filtrados
  const [readAlertIds, setReadAlertIds] = useState<Set<string>>(new Set());

  const filteredAlerts = useMemo(() => {
    return alerts.filter((a) => {
      const isRead = readAlertIds.has(a.id);
      if (alertFilter === "UNREAD" && isRead) return false;

      if (searchFilter.trim()) {
        const query = normalizeText(searchFilter.trim());
        const matchesTitle = normalizeText(a.title).includes(query);
        const matchesSnippet = normalizeText(a.snippet).includes(query);
        const matchesLabel = normalizeText(a.targetLabel).includes(query);
        const matchesValue = normalizeText(a.targetValue).includes(query);
        return matchesTitle || matchesSnippet || matchesLabel || matchesValue;
      }

      return true;
    });
  }, [alerts, readAlertIds, alertFilter, searchFilter]);

  const unreadCount = alerts.filter((a) => !readAlertIds.has(a.id)).length;

  function handleAddTarget(e: React.FormEvent) {
    e.preventDefault();
    if (!newValue.trim()) {
      showToast({ title: "Valor obrigatório", description: "Digite o CPF, Matrícula ou Termo a ser monitorado.", tone: "danger" });
      return;
    }

    const newTarget: MonitoredTarget = {
      id: `target-${Date.now()}`,
      type: newType,
      value: newValue.trim(),
      label: newLabel.trim() || `Monitoramento de ${newType}: ${newValue.trim()}`,
      active: true,
      notifyOnNomeacao,
      notifyOnExoneracao,
      notifyOnRelotacao,
      notifyOnContrato,
      createdAt: new Date().toISOString(),
    };

    setTargets((prev) => [newTarget, ...prev]);
    setNewValue("");
    setNewLabel("");
    showToast({ title: "Alvo Adicionado ao Monitoramento", description: `Agora o Projeto Nova monitora continuamente: ${newTarget.value}` });
  }

  function handleToggleTarget(id: string) {
    setTargets((prev) =>
      prev.map((t) => (t.id === id ? { ...t, active: !t.active } : t))
    );
    showToast({ title: "Status do Monitoramento Atualizado", description: "Configuração salva com sucesso." });
  }

  function handleDeleteTarget(id: string) {
    setTargets((prev) => prev.filter((t) => t.id !== id));
    showToast({ title: "Alvo Removido", description: "O item não será mais acompanhado nas próximas edições." });
  }

  function handleMarkAllRead() {
    const allIds = new Set(alerts.map((a) => a.id));
    setReadAlertIds(allIds);
    showToast({ title: "Alertas Marcados como Lidos", description: `${unreadCount} alerta(s) atualizado(s).` });
  }

  function handleToggleRead(alertId: string) {
    setReadAlertIds((prev) => {
      const next = new Set(prev);
      if (next.has(alertId)) {
        next.delete(alertId);
      } else {
        next.add(alertId);
      }
      return next;
    });
  }

  function handleTriggerTestAlert(target: MonitoredTarget) {
    showToast({
      title: `🔔 Teste de Notificação Instantânea (${target.type})`,
      description: `Alerta ativo para ${target.label} (${target.value}). ${alertCountByTarget[target.id] || 0} ocorrência(s) encontrada(s) nos diários.`,
    });
  }

  return (
    <div className="flex flex-col gap-8">
      {/* Cabeçalho do Centro de Monitoramento */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
        <div className="flex items-start gap-3">
          <div className="rounded-lg bg-primary/10 p-2.5 text-primary">
            <Bell size={24} />
          </div>
          <div>
            <h1 className="text-xl font-bold text-foreground">Centro de Monitoramento & Alertas por CPF e Matrícula</h1>
            <p className="text-xs text-muted mt-0.5">
              Cadastre CPFs, Matrículas Funcionais, Nomes e CNPJs para receber alertas automáticos toda vez que forem citados em edições do Diário Oficial.
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex flex-col items-end">
            <span className="text-xs text-muted font-medium">Alvos Monitorados:</span>
            <strong className="text-sm font-bold text-primary">{targets.filter((t) => t.active).length} Ativos</strong>
          </div>
          <div className="h-8 w-px bg-surface-border" />
          <div className="flex flex-col items-end">
            <span className="text-xs text-muted font-medium">Alertas Não Lidos:</span>
            <span className="inline-flex items-center gap-1 text-sm font-bold text-danger">
              <AlertTriangle size={14} />
              {unreadCount}
            </span>
          </div>
        </div>
      </div>

      {/* Formulário para Adicionar Novo Alvo de Monitoramento */}
      <form onSubmit={handleAddTarget} className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
        <div className="flex items-center justify-between border-b border-surface-border pb-3">
          <h2 className="text-sm font-bold text-foreground flex items-center gap-2">
            <Plus size={16} className="text-primary" />
            Cadastrar Novo Alvo para Monitoramento Contínuo
          </h2>
          <span className="text-xs text-muted font-mono">Formato Automático Normalizado</span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-foreground">Tipo do Alvo:</label>
            <select
              value={newType}
              onChange={(e) => setNewType(e.target.value as TargetType)}
              className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs font-semibold text-foreground"
            >
              <option value="CPF">👤 CPF do Servidor / Pessoa Física</option>
              <option value="MATRICULA">📋 Matrícula Funcional</option>
              <option value="NOME">📛 Nome Completo / Servidor Público</option>
              <option value="CNPJ">🏢 CNPJ da Empresa / Fornecedor</option>
              <option value="TERMO_GERAL">🔍 Termo Geral / Palavra-Chave</option>
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-foreground">Valor a Monitorar (CPF, Matrícula ou Termo):</label>
            <Input
              placeholder={
                newType === "CPF"
                  ? "021.946.881-88 ou 02194688188"
                  : newType === "MATRICULA"
                  ? "MAT-2025-03 ou 10492"
                  : newType === "CNPJ"
                  ? "03.492.110/0001-99"
                  : "Digite o texto exato..."
              }
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              className="text-xs"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-foreground">Rótulo / Identificador Personalizado:</label>
            <Input
              placeholder="Ex: Auditoria Gabinete do Prefeito"
              value={newLabel}
              onChange={(e) => setNewLabel(e.target.value)}
              className="text-xs"
            />
          </div>
        </div>

        {/* Gatilhos de Alerta */}
        <div className="flex flex-wrap items-center justify-between gap-4 pt-3 border-t border-surface-border text-xs">
          <span className="font-semibold text-muted">Gatilhos de Notificação Ativos:</span>
          <div className="flex flex-wrap items-center gap-4">
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyOnNomeacao}
                onChange={(e) => setNotifyOnNomeacao(e.target.checked)}
                className="rounded border-surface-border text-primary"
              />
              <span>Nomeação / Posse</span>
            </label>
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyOnExoneracao}
                onChange={(e) => setNotifyOnExoneracao(e.target.checked)}
                className="rounded border-surface-border text-primary"
              />
              <span>Exoneração / Desligamento</span>
            </label>
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyOnRelotacao}
                onChange={(e) => setNotifyOnRelotacao(e.target.checked)}
                className="rounded border-surface-border text-primary"
              />
              <span>Relotação / Transferência</span>
            </label>
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyOnContrato}
                onChange={(e) => setNotifyOnContrato(e.target.checked)}
                className="rounded border-surface-border text-primary"
              />
              <span>Fiscal de Contrato</span>
            </label>
          </div>

          <Button type="submit" size="sm" variant="primary" className="ml-auto">
            <Plus size={14} className="mr-1" />
            Salvar e Iniciar Acompanhamento
          </Button>
        </div>
      </form>

      {/* Lista de Alvos Salvos para Observação */}
      <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
        <div className="flex items-center justify-between border-b border-surface-border pb-3">
          <div>
            <h2 className="text-base font-bold text-foreground flex items-center gap-2">
              <ShieldCheck size={18} className="text-primary" />
              Lista de Alvos em Acompanhamento Continuo ({targets.length})
            </h2>
            <p className="text-xs text-muted">Estes itens são varridos automaticamente em cada nova publicação capturada pelo pipeline.</p>
          </div>
        </div>

        {targets.length === 0 ? (
          <EmptyState
            title="Nenhum alvo cadastrado"
            description="Utilize o formulário acima para adicionar CPFs, Matrículas ou Nomes que deseja monitorar."
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {targets.map((t) => {
              const occurrences = alertCountByTarget[t.id] || 0;

              return (
                <div
                  key={t.id}
                  className={`flex flex-col justify-between gap-3 rounded-lg border p-4 transition-all shadow-sm ${
                    t.active
                      ? "border-surface-border bg-surface hover:border-primary/40"
                      : "border-surface-border/50 bg-surface-border/10 opacity-60"
                  }`}
                >
                  <div className="flex flex-col gap-2">
                    <div className="flex items-center justify-between gap-2 border-b border-surface-border/60 pb-2">
                      <div className="flex items-center gap-2">
                        <Badge tone={t.type === "CPF" ? "success" : t.type === "MATRICULA" ? "warning" : t.type === "CNPJ" ? "neutral" : "info"}>
                          {t.type}
                        </Badge>
                        <span className="text-xs font-bold font-mono text-foreground">{t.value}</span>
                      </div>
                      <Badge tone={t.active ? "success" : "neutral"} className="text-[10px]">
                        {t.active ? "Ativo" : "Pausado"}
                      </Badge>
                    </div>

                    <h3 className="text-sm font-semibold text-foreground line-clamp-1">{t.label}</h3>

                    <div className="flex flex-wrap gap-1 text-[11px] text-muted">
                      {t.notifyOnNomeacao && <span className="bg-success/10 text-success px-1.5 py-0.5 rounded">Nomeação</span>}
                      {t.notifyOnExoneracao && <span className="bg-danger/10 text-danger px-1.5 py-0.5 rounded">Exoneração</span>}
                      {t.notifyOnRelotacao && <span className="bg-warning/10 text-warning px-1.5 py-0.5 rounded">Relotação</span>}
                      {t.notifyOnContrato && <span className="bg-primary/10 text-primary px-1.5 py-0.5 rounded">Contratos</span>}
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-3 border-t border-surface-border/60 text-xs">
                    <div className="flex items-center gap-1.5 text-muted">
                      <Bell size={13} className={occurrences > 0 ? "text-danger" : "text-muted"} />
                      <span>
                        <strong className="text-foreground">{occurrences}</strong> ocorrência(s)
                      </span>
                    </div>

                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleTriggerTestAlert(t)}
                        title="Testar Notificação Instantânea"
                        className="rounded p-1.5 text-muted hover:bg-surface-border hover:text-foreground"
                      >
                        <Bell size={14} />
                      </button>
                      <button
                        onClick={() => handleToggleTarget(t.id)}
                        title={t.active ? "Pausar Monitoramento" : "Ativar Monitoramento"}
                        className="rounded p-1.5 text-muted hover:bg-surface-border hover:text-foreground"
                      >
                        {t.active ? <Pause size={14} /> : <Play size={14} />}
                      </button>
                      <button
                        onClick={() => handleDeleteTarget(t.id)}
                        title="Remover Alvo"
                        className="rounded p-1.5 text-danger hover:bg-danger/10"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Feed de Alertas & Notificações de Auditoria */}
      <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-surface-border pb-3">
          <div>
            <h2 className="text-base font-bold text-foreground flex items-center gap-2">
              <AlertTriangle size={18} className="text-danger" />
              Feed de Alertas e Ocorrências Detectadas ({filteredAlerts.length})
            </h2>
            <p className="text-xs text-muted">Notificações geradas automaticamente a partir dos CPFs e Matrículas cadastrados.</p>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant={alertFilter === "ALL" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setAlertFilter("ALL")}
            >
              Todos ({alerts.length})
            </Button>
            <Button
              variant={alertFilter === "UNREAD" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setAlertFilter("UNREAD")}
            >
              Não Lidos ({unreadCount})
            </Button>
            {unreadCount > 0 && (
              <Button variant="secondary" size="sm" onClick={handleMarkAllRead}>
                <Check size={14} className="mr-1" />
                Marcar Todos como Lidos
              </Button>
            )}
          </div>
        </div>

        {/* Filtro textual de busca de alertas */}
        <div className="relative">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted" />
          <Input
            placeholder="Filtrar alertas por texto, nome, CPF ou título..."
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="pl-9 text-xs"
          />
        </div>

        {/* Lista de Alertas */}
        {filteredAlerts.length === 0 ? (
          <EmptyState
            title="Nenhuma ocorrência encontrada"
            description="Não há alertas pendentes para os alvos cadastrados na busca atual."
          />
        ) : (
          <div className="flex flex-col gap-3">
            {filteredAlerts.map((a) => {
              const isRead = readAlertIds.has(a.id);

              return (
                <div
                  key={a.id}
                  className={`flex flex-col gap-2.5 rounded-lg border p-4 transition-all shadow-sm ${
                    isRead
                      ? "border-surface-border/60 bg-surface/50 opacity-80"
                      : "border-primary/40 bg-primary/5 hover:border-primary/60"
                  }`}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/50 pb-2">
                    <div className="flex flex-wrap items-center gap-2">
                      {!isRead && <span className="h-2 w-2 rounded-full bg-danger animate-pulse" />}
                      <Badge tone={a.targetType === "CPF" ? "success" : a.targetType === "MATRICULA" ? "warning" : "info"}>
                        Alvo: {a.targetLabel} ({a.targetValue})
                      </Badge>
                      {a.editionNumber && (
                        <span className="text-xs text-muted font-medium">Edição Nº {a.editionNumber}</span>
                      )}
                    </div>

                    <span className="text-xs text-muted font-mono">
                      {a.createdAt ? new Date(a.createdAt).toLocaleDateString("pt-BR") : "Recente"}
                    </span>
                  </div>

                  <h4 className="text-sm font-bold text-foreground">{a.title}</h4>
                  <p className="text-xs text-foreground/90 bg-surface p-2.5 rounded border border-surface-border/50 italic leading-relaxed">
                    &ldquo;{a.snippet}&rdquo;
                  </p>

                  <div className="flex items-center justify-between pt-1 text-xs">
                    <button
                      onClick={() => handleToggleRead(a.id)}
                      className="inline-flex items-center gap-1 font-medium text-muted hover:text-foreground"
                    >
                      <CheckCircle2 size={14} className={isRead ? "text-success" : "text-muted"} />
                      {isRead ? "Marcar como Não Lido" : "Marcar como Lido"}
                    </button>

                    <a
                      href={a.docUrl || "https://www.rondonopolis.mt.gov.br/diario-oficial/"}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 font-medium text-primary hover:underline"
                    >
                      <ExternalLink size={13} />
                      Abrir PDF do Diário Oficial →
                    </a>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
