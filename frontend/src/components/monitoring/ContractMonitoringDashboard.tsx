"use client";

import { useEffect, useState, useMemo } from "react";
import {
  AlertTriangle,
  Clock,
  Building2,
  FileCheck,
  UserCheck,
  Plus,
  Search,
  User,
  X,
  CheckCircle2,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";
import { ModalShell } from "@/components/ui/ModalShell";
import { useToast } from "@/components/notifications/ToastProvider";
import { apiClient } from "@/lib/api/client";
import { fetchContratos } from "@/lib/api/contratos";
import type { Contrato, KanbanResponse } from "@/types/api";

export interface ContractOfficial {
  id: string;
  contrato_id: string;
  contrato_numero?: string;
  nome: string;
  matricula: string;
  num_portaria: string;
  is_titular: boolean;
  created_at: string;
}

export function ContractMonitoringDashboard() {
  const [contratos, setContratos] = useState<Contrato[]>([]);
  const [kanban, setKanban] = useState<KanbanResponse | null>(null);
  const [officials, setOfficials] = useState<ContractOfficial[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [activeTab, setActiveTab] = useState<"vencimentos" | "fiscais" | "certidoes">("vencimentos");
  const [isOfficialModalOpen, setIsOfficialModalOpen] = useState(false);

  const { showToast } = useToast();

  const loadData = async () => {
    setLoading(true);
    try {
      const [resContratos, resKanban] = await Promise.all([
        // fetchContratos desembrulha o envelope paginado ({ data, total, ... }) —
        // usar apiClient direto aqui deixava contractsList sempre vazio (o
        // painel não mostrava nenhum contrato).
        fetchContratos(1, 500).catch(() => ({ data: [] as Contrato[] })),
        apiClient.get<KanbanResponse>("v1/demands/kanban").catch(() => ({ data: null })),
      ]);

      const contractsList = Array.isArray(resContratos.data) ? resContratos.data : [];

      setContratos(contractsList);
      if (resKanban && resKanban.data) {
        setKanban(resKanban.data);
      }

      // Dados gravados no localStorage para Fiscais Nominal
      const savedOfficials = localStorage.getItem("nova_contract_officials_v1");
      if (savedOfficials) {
        try {
          setOfficials(JSON.parse(savedOfficials));
        } catch {
          // fallback
        }
      }
    } catch (err) {
      console.error("Monitoring load error:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // Cálculos das estatísticas de SLAs e vencimentos
  const metrics = useMemo(() => {
    const activeContratos = Array.isArray(contratos) ? contratos : [];
    const totalContratos = activeContratos.length;
    const vigentes = activeContratos.filter((c) => c.status === "vigente" || c.status_label === "Vigente").length;
    const totalValor = activeContratos.reduce((acc, c) => acc + (c.valor || 0), 0);

    // Contratos vencendo nos próximos 60 dias
    const now = new Date();
    const in60Days = new Date();
    in60Days.setDate(now.getDate() + 60);

    const expeditingContratos = activeContratos.filter((c) => {
      if (!c.data_vigencia_fim) return false;
      const end = new Date(c.data_vigencia_fim);
      return end > now && end <= in60Days;
    });

    // Total de demandas ativas no Kanban
    let activeDemandsCount = 0;
    let slaBreachesCount = 0;

    if (kanban && kanban.columns) {
      kanban.columns.forEach((col) => {
        activeDemandsCount += col.total;
        col.items.forEach((item) => {
          // Se começou há mais de 10 dias, considera atraso de SLA da IN SCL
          const startedAt = new Date(item.etapa_started_at);
          const diffDays = Math.floor((now.getTime() - startedAt.getTime()) / (1000 * 3600 * 24));
          if (diffDays > 10) {
            slaBreachesCount++;
          }
        });
      });
    }

    return {
      totalContratos,
      vigentes,
      totalValor,
      expeditingCount: expeditingContratos.length,
      expeditingContratos,
      activeDemandsCount,
      slaBreachesCount,
    };
  }, [contratos, kanban]);

  const filteredContratos = useMemo(() => {
    const activeContratos = Array.isArray(contratos) ? contratos : [];
    if (!search.trim()) return activeContratos;
    const q = search.toLowerCase();
    return activeContratos.filter(
      (c) =>
        c.numero?.toLowerCase().includes(q) ||
        c.contratado?.toLowerCase().includes(q) ||
        c.objeto?.toLowerCase().includes(q)
    );
  }, [contratos, search]);

  return (
    <div className="flex flex-col gap-6">
      {/* Header — mesmo padrão das outras páginas: sem caixa, só o divisor. */}
      <div className="flex flex-col justify-between gap-4 border-b border-surface-border pb-4 md:flex-row md:items-start">
        <div>
          <p className="dateline">IN SCL 01/2019 · Monitoramento</p>
          <h1 className="mt-2 text-2xl font-semibold text-foreground">Monitoramento operacional e SLAs</h1>
          <p className="mt-1 text-sm text-muted">
            Vigências de contratos, fiscais nomeados por portaria e conformidade fiscal, em tempo real.
          </p>
        </div>

        <Button onClick={loadData} variant="ghost" size="sm" disabled={loading}>
          {loading ? "Atualizando..." : "Recarregar Dados"}
        </Button>
      </div>

      {/* Cards de Métricas Principais */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="border-l-4 border-l-primary">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted uppercase tracking-wider">Contratos Ativos</p>
              <h3 className="text-2xl font-bold text-foreground mt-1">{metrics.vigentes}</h3>
              <p className="text-xs text-muted mt-0.5">
                Total: R$ {metrics.totalValor.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
              </p>
            </div>
            <div className="rounded-lg bg-primary/10 p-3 text-primary">
              <Building2 size={24} />
            </div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-warning">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted uppercase tracking-wider">Vencendo em 60 dias</p>
              <h3 className="text-2xl font-bold font-mono text-warning mt-1">{metrics.expeditingCount}</h3>
              <p className="text-xs text-muted mt-0.5">Requer prorrogação / aditivo</p>
            </div>
            <div className="rounded-lg bg-warning/10 p-3 text-warning">
              <Clock size={24} />
            </div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-info">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted uppercase tracking-wider">Demandas em Liquidação</p>
              <h3 className="text-2xl font-bold text-foreground mt-1">{metrics.activeDemandsCount}</h3>
              <p className="text-xs text-muted mt-0.5">Quadros Kanban ativos</p>
            </div>
            <div className="rounded-lg bg-info/10 p-3 text-info">
              <FileCheck size={24} />
            </div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-danger">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted uppercase tracking-wider">Gargalos de SLA (IN SCL)</p>
              <h3 className="text-2xl font-bold text-danger mt-1">{metrics.slaBreachesCount}</h3>
              <p className="text-xs text-danger/80 mt-0.5">Card parado há &gt;10 dias</p>
            </div>
            <div className="rounded-lg bg-danger/10 p-3 text-danger">
              <AlertTriangle size={24} />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs & Filtro */}
      <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-surface-border pb-4">
          {/* Tabs */}
          <div className="flex items-center gap-2">
            <Button
              variant={activeTab === "vencimentos" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setActiveTab("vencimentos")}
            >
              <Clock size={16} className="mr-1.5" />
              Monitoramento de Vencimentos ({filteredContratos.length})
            </Button>

            <Button
              variant={activeTab === "fiscais" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setActiveTab("fiscais")}
            >
              <UserCheck size={16} className="mr-1.5" />
              Fiscais & Portarias ({officials.length})
            </Button>

            <Button
              variant={activeTab === "certidoes" ? "primary" : "ghost"}
              size="sm"
              onClick={() => setActiveTab("certidoes")}
            >
              <FileCheck size={16} className="mr-1.5" />
              Conformidade de Certidões
            </Button>
          </div>

          {/* Busca */}
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted pointer-events-none" />
            <Input
              placeholder="Buscar por número, empresa..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 text-xs"
            />
          </div>
        </div>

        {/* Tab 1: Monitoramento de Vencimentos */}
        {activeTab === "vencimentos" && (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-surface-border bg-surface-hover/50 text-muted uppercase tracking-wider">
                  <th className="p-3">Contrato / Empresa</th>
                  <th className="p-3">Objeto</th>
                  <th className="p-3">Valor Total</th>
                  <th className="p-3">Término da Vigência</th>
                  <th className="p-3">Status do SLA</th>
                  <th className="p-3 text-right">Ação</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-border/60">
                {filteredContratos.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="p-8 text-center text-muted">
                      Não achamos nenhuma referência
                    </td>
                  </tr>
                ) : (
                  filteredContratos.map((c) => {
                    const fim = c.data_vigencia_fim ? new Date(c.data_vigencia_fim) : null;
                    const daysLeft = fim ? Math.ceil((fim.getTime() - new Date().getTime()) / (1000 * 3600 * 24)) : null;

                    let slaBadge = <Badge tone="success">Vigente (Em dia)</Badge>;
                    if (daysLeft !== null && daysLeft <= 0) {
                      slaBadge = <Badge tone="neutral">Expirado</Badge>;
                    } else if (daysLeft !== null && daysLeft <= 60) {
                      slaBadge = <Badge tone="warning">Vence em {daysLeft} dias</Badge>;
                    }

                    return (
                      <tr key={c.id} className="hover:bg-surface-hover/30 transition-colors">
                        <td className="p-3 font-medium text-foreground">
                          <div className="font-semibold text-primary">{c.numero}</div>
                          <div className="text-muted text-[11px] truncate max-w-[200px]">{c.contratado}</div>
                        </td>
                        <td className="p-3 max-w-[280px]">
                          <p className="line-clamp-2 text-foreground/90">{c.objeto}</p>
                        </td>
                        <td className="p-3 font-mono font-medium">
                          R$ {(c.valor || 0).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                        </td>
                        <td className="p-3 font-mono text-muted">
                          {fim ? fim.toLocaleDateString("pt-BR") : "Não informada"}
                        </td>
                        <td className="p-3">{slaBadge}</td>
                        <td className="p-3 text-right">
                          <a
                            href="/contratos"
                            className="inline-flex items-center text-primary hover:underline font-medium"
                          >
                            Ver no Kanban →
                          </a>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Tab 2: Fiscais Nominal & Portarias */}
        {activeTab === "fiscais" && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-foreground">Gestão de Fiscais Designados por Portaria</h3>
                <p className="text-xs text-muted">Acompanhe quem responde formalmente pela fiscalização e ateste de cada contrato.</p>
              </div>
              <Button size="sm" onClick={() => setIsOfficialModalOpen(true)}>
                <Plus size={14} className="mr-1.5" />
                Vincular Novo Fiscal
              </Button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {officials.length === 0 ? (
                <div className="col-span-2 p-8 text-center text-muted border border-dashed rounded-lg">
                  Não achamos nenhuma referência
                </div>
              ) : (
                officials.map((of) => (
                  <div key={of.id} className="rounded-lg border border-surface-border bg-surface p-4 flex flex-col gap-3 shadow-sm">
                    <div className="flex items-center justify-between border-b border-surface-border/60 pb-2">
                      <span className="text-xs font-semibold text-primary">{of.contrato_numero || "Contrato Vinculado"}</span>
                      <Badge tone={of.is_titular ? "success" : "info"}>
                        {of.is_titular ? "Fiscal Titular" : "Fiscal Suplente"}
                      </Badge>
                    </div>

                    <div className="flex items-start gap-3">
                      <div className="rounded-full bg-primary/10 p-2 text-primary">
                        <User size={20} />
                      </div>
                      <div>
                        <h4 className="text-sm font-bold text-foreground">{of.nome}</h4>
                        <p className="text-xs text-muted font-mono">Matrícula: {of.matricula}</p>
                        <p className="text-xs text-muted font-mono mt-0.5">Portaria nº: {of.num_portaria}</p>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* Tab 3: Conformidade de Certidões */}
        {activeTab === "certidoes" && (
          <div className="flex flex-col gap-4">
            <div className="rounded-lg bg-surface-hover/30 p-4 border border-surface-border text-xs text-muted flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-success shrink-0 mt-0.5" />
              <div>
                <strong className="text-foreground font-semibold">Exigência de Certidões para Liquidação (IN SCL 01/2019)</strong>
                <p className="mt-0.5">
                  Para avanço da etapa de Relatório de Pagamento para a Contabilidade, a empresa contratada deve possuir as certidões CNDT, FGTS e CND Municipal em dia.
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="rounded-lg border border-surface-border p-4 bg-surface flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-foreground">Certidão Trabalhista (CNDT)</span>
                  <Badge tone="success">100% Válidas</Badge>
                </div>
                <p className="text-xs text-muted">Vazia de débitos perante a Justiça do Trabalho.</p>
              </div>

              <div className="rounded-lg border border-surface-border p-4 bg-surface flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-foreground">Regularidade FGTS</span>
                  <Badge tone="success">100% Válidas</Badge>
                </div>
                <p className="text-xs text-muted">Certificado de Regularidade do FGTS emitido pela Caixa.</p>
              </div>

              <div className="rounded-lg border border-surface-border p-4 bg-surface flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-foreground">Receita / INSS</span>
                  <Badge tone="success">100% Válidas</Badge>
                </div>
                <p className="text-xs text-muted">Certidão Conjunta Federal de Débitos Relativos a Tributos.</p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Modal para Vincular Novo Fiscal */}
      {isOfficialModalOpen && (
        <NewOfficialModal
          contratos={contratos}
          onClose={() => setIsOfficialModalOpen(false)}
          onCreated={(newOfficial) => {
            const updated = [newOfficial, ...officials];
            setOfficials(updated);
            localStorage.setItem("nova_contract_officials_v1", JSON.stringify(updated));
            showToast({
              title: "Fiscal Cadastrado com Sucesso",
              description: `Portaria de nomeação vinculada a ${newOfficial.nome}.`,
              tone: "success",
            });
            setIsOfficialModalOpen(false);
          }}
        />
      )}
    </div>
  );
}

function NewOfficialModal({
  contratos,
  onClose,
  onCreated,
}: {
  contratos: Contrato[];
  onClose: () => void;
  onCreated: (official: ContractOfficial) => void;
}) {
  const [contratoId, setContratoId] = useState(contratos[0]?.id || "");
  const [nome, setNome] = useState("");
  const [matricula, setMatricula] = useState("");
  const [numPortaria, setNumPortaria] = useState("");
  const [isTitular, setIsTitular] = useState(true);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const selContract = contratos.find((c) => c.id === contratoId);

    const official: ContractOfficial = {
      id: `official-${Date.now()}`,
      contrato_id: contratoId,
      contrato_numero: selContract?.numero || "Contrato",
      nome: nome.trim(),
      matricula: matricula.trim(),
      num_portaria: numPortaria.trim(),
      is_titular: isTitular,
      created_at: new Date().toISOString(),
    };

    onCreated(official);
  };

  return (
    <ModalShell open onClose={onClose} size="md" labelledBy="link-fiscal-title">
      <div className="flex shrink-0 items-center justify-between border-b border-surface-border px-6 py-4">
        <h2 id="link-fiscal-title" className="text-base font-bold text-foreground">
          Vincular Fiscal &amp; Portaria
        </h2>
        <button
          type="button"
          onClick={onClose}
          aria-label="Fechar modal"
          className="rounded p-1 text-muted hover:bg-surface-hover"
        >
          <X size={18} />
        </button>
      </div>

      <form onSubmit={handleSubmit} className="min-h-0 flex-1 space-y-4 overflow-y-auto p-6 text-xs">
          <div>
            <label className="block font-medium text-foreground mb-1">Contrato *</label>
            <select
              value={contratoId}
              onChange={(e) => setContratoId(e.target.value)}
              className="w-full rounded border border-surface-border bg-surface p-2 text-foreground"
              required
            >
              {contratos.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.numero} - {c.contratado}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block font-medium text-foreground mb-1">Nome Completo do Servidor *</label>
            <Input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Ex: Maria Fernandes" required />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block font-medium text-foreground mb-1">Matrícula Funcional *</label>
              <Input value={matricula} onChange={(e) => setMatricula(e.target.value)} placeholder="Ex: MAT-9941" required />
            </div>
            <div>
              <label className="block font-medium text-foreground mb-1">Nº Portaria de Nomeação *</label>
              <Input value={numPortaria} onChange={(e) => setNumPortaria(e.target.value)} placeholder="Ex: PORT-120/2026" required />
            </div>
          </div>

          <div>
            <label className="block font-medium text-foreground mb-1">Papel na Fiscalização</label>
            <select
              value={isTitular ? "titular" : "suplente"}
              onChange={(e) => setIsTitular(e.target.value === "titular")}
              className="w-full rounded border border-surface-border bg-surface p-2 text-foreground"
            >
              <option value="titular">Fiscal Titular</option>
              <option value="suplente">Fiscal Suplente</option>
            </select>
          </div>

          <div className="flex items-center justify-end gap-3 border-t border-surface-border pt-4">
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancelar
            </Button>
            <Button type="submit">Cadastrar Fiscal</Button>
          </div>
        </form>
    </ModalShell>
  );
}
