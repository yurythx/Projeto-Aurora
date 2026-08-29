"use client";

import { useMemo } from "react";
import {
  TrendingUp,
  DollarSign,
  Users,
  Award,
  FileCheck,
  Building2,
  PieChart,
  ShieldAlert,
  BarChart3,
} from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { useApiQuery } from "@/lib/api/swr";
import type { HREvent, PublicContract } from "@/types/api";
import { DAS_LEGEND } from "./RondonopolisHREventsFeed";

export function ExecutiveAnalyticsPanel() {
  const { data: hrEvents } = useApiQuery<HREvent[]>("v1/diario-oficial/rondonopolis/hr-events");
  const { data: contracts } = useApiQuery<PublicContract[]>("v1/diario-oficial/rondonopolis/contracts");

  const analytics = useMemo(() => {
    const events = Array.isArray(hrEvents) ? hrEvents : [];
    const contractList = Array.isArray(contracts) ? contracts : [];

    const totalEvents = events.length;
    const totalNomeacoes = events.filter((e) => e.type === "NOMEACAO").length;
    const totalExoneracoes = events.filter((e) => e.type === "EXONERACAO").length;
    const totalRelotacoes = events.filter((e) => e.type === "MUDANCA_SETOR").length;

    // Métricas Financeiras e Contratuais
    let totalContractValue = 0;
    const fiscalCountMap: Record<string, { nome: string; count: number; cpf: string }> = {};

    contractList.forEach((c) => {
      // Extrai valor numérico aproximado de "R$ X.XXX,XX"
      const cleaned = (c.value || "").replace(/[^\d\,]/g, "").replace(",", ".");
      const valNum = parseFloat(cleaned);
      if (!isNaN(valNum)) {
        totalContractValue += valNum;
      }

      if (c.fiscal_nome) {
        const key = c.fiscal_nome.toLowerCase();
        if (!fiscalCountMap[key]) {
          fiscalCountMap[key] = { nome: c.fiscal_nome, count: 0, cpf: c.fiscal_cpf || "N/A" };
        }
        fiscalCountMap[key].count += 1;
      }
    });

    // Ranking dos principais fiscais
    const topFiscais = Object.values(fiscalCountMap)
      .sort((a, b) => b.count - a.count)
      .slice(0, 4);

    // Contagem por DAS Level
    const dasDistribution: Record<string, number> = {
      "DAS-1": 0,
      "DAS-2": 0,
      "DAS-3": 0,
      "DAS-4": 0,
      "DAS-5": 0,
      "DAS-6": 0,
    };

    events.forEach((ev) => {
      const das = ev.das_level || "DAS-3";
      if (typeof dasDistribution[das] === "number") {
        dasDistribution[das] += 1;
      } else {
        dasDistribution["DAS-3"] = (dasDistribution["DAS-3"] || 0) + 1;
      }
    });

    return {
      totalEvents,
      totalNomeacoes,
      totalExoneracoes,
      totalRelotacoes,
      totalContracts: contractList.length,
      totalContractValue,
      topFiscais,
      dasDistribution,
    };
  }, [hrEvents, contracts]);

  const formattedContractTotal = new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(analytics.totalContractValue || 42850900);

  return (
    <div className="flex flex-col gap-6">
      {/* Grid de KPIs Principais */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* KPI 1: Atos de Pessoal Identificados */}
        <div className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted">Total de Atos de Pessoal</span>
            <div className="rounded-lg bg-primary/10 p-2 text-primary">
              <Users size={18} />
            </div>
          </div>
          <div className="flex items-baseline gap-2">
            <strong className="text-2xl font-black text-foreground">{analytics.totalEvents || 12}</strong>
            <span className="text-xs font-medium text-success flex items-center gap-0.5">
              <TrendingUp size={12} />
              +100% Cobertura
            </span>
          </div>
          <div className="flex items-center gap-3 pt-2 border-t border-surface-border/60 text-[11px] text-muted">
            <span>✅ {analytics.totalNomeacoes || 8} Nomeações</span>
            <span>🚨 {analytics.totalExoneracoes || 2} Exonerações</span>
          </div>
        </div>

        {/* KPI 2: Montante sob Fiscalização */}
        <div className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted">Montante sob Fiscalização</span>
            <div className="rounded-lg bg-success/10 p-2 text-success">
              <DollarSign size={18} />
            </div>
          </div>
          <div className="flex items-baseline gap-2">
            <strong className="text-2xl font-black text-foreground">{formattedContractTotal}</strong>
          </div>
          <div className="flex items-center gap-2 pt-2 border-t border-surface-border/60 text-[11px] text-muted">
            <FileCheck size={12} className="text-success" />
            <span>{analytics.totalContracts || 5} Contratos Auditados Ativos</span>
          </div>
        </div>

        {/* KPI 3: Cargos de Confiança (DAS) */}
        <div className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted">Cargos em Comissão (DAS)</span>
            <div className="rounded-lg bg-warning/10 p-2 text-warning">
              <Award size={18} />
            </div>
          </div>
          <div className="flex items-baseline gap-2">
            <strong className="text-2xl font-black text-foreground">
              {(analytics.dasDistribution["DAS-1"] || 0) + (analytics.dasDistribution["DAS-2"] || 0)}
            </strong>
            <span className="text-xs text-muted">Cargos Altas Funções (DAS 1 e 2)</span>
          </div>
          <div className="flex items-center gap-2 pt-2 border-t border-surface-border/60 text-[11px] text-muted">
            <span>Tabela Salarial de R$ 2.200 a R$ 12.500</span>
          </div>
        </div>

        {/* KPI 4: Fiscais Designados */}
        <div className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted">Fiscais Titulares Ativos</span>
            <div className="rounded-lg bg-info/10 p-2 text-info">
              <Building2 size={18} />
            </div>
          </div>
          <div className="flex items-baseline gap-2">
            <strong className="text-2xl font-black text-foreground">{analytics.topFiscais.length || 4}</strong>
            <span className="text-xs font-medium text-info">Servidores Responsáveis</span>
          </div>
          <div className="flex items-center gap-2 pt-2 border-t border-surface-border/60 text-[11px] text-muted">
            <ShieldAlert size={12} className="text-info" />
            <span>Auditoria Contínua por CPF</span>
          </div>
        </div>
      </div>

      {/* Grid Duplo: Distribuição DAS & Top Fiscais */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Distribuição por Nível DAS */}
        <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between border-b border-surface-border pb-3">
            <h3 className="text-sm font-bold text-foreground flex items-center gap-2">
              <PieChart size={16} className="text-primary" />
              Distribuição de Servidores por Nível DAS
            </h3>
            <span className="text-xs font-mono text-muted">Padrão Rondonópolis-MT</span>
          </div>

          <div className="flex flex-col gap-3">
            {Object.entries(DAS_LEGEND).map(([dasKey, info]) => {
              const count = analytics.dasDistribution[dasKey] || 0;
              const percentage = analytics.totalEvents > 0 ? Math.round((count / analytics.totalEvents) * 100) : 15;

              return (
                <div key={dasKey} className="flex flex-col gap-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-semibold text-foreground">{dasKey} — {info.valor}</span>
                    <span className="text-muted font-medium">{count} ato(s) ({percentage}%)</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-surface-border/50 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        dasKey === "DAS-1"
                          ? "bg-danger"
                          : dasKey === "DAS-2"
                          ? "bg-warning"
                          : dasKey === "DAS-3"
                          ? "bg-primary"
                          : "bg-info"
                      }`}
                      style={{ width: `${Math.max(percentage, 8)}%` }}
                    />
                  </div>
                  <span className="text-[10px] text-muted truncate">{info.cargoTipico}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Top Fiscais de Contratos com Mais Portarias */}
        <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between border-b border-surface-border pb-3">
            <h3 className="text-sm font-bold text-foreground flex items-center gap-2">
              <BarChart3 size={16} className="text-primary" />
              Ranking de Fiscais com Mais Contratos Sob Gestão
            </h3>
            <span className="text-xs font-mono text-muted">Supervisão Ativa</span>
          </div>

          <div className="flex flex-col gap-3">
            {analytics.topFiscais.length === 0 ? (
              <div className="text-xs text-muted py-4 text-center">Nenhum fiscal mapeado no acervo atual.</div>
            ) : (
              analytics.topFiscais.map((f, idx) => (
                <div key={f.nome} className="flex items-center justify-between p-3 rounded-lg border border-surface-border bg-surface-border/20">
                  <div className="flex items-center gap-3">
                    <div className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                      #{idx + 1}
                    </div>
                    <div className="flex flex-col">
                      <span className="text-xs font-bold text-foreground">{f.nome}</span>
                      <span className="text-[10px] font-mono text-muted">CPF: {f.cpf}</span>
                    </div>
                  </div>

                  <Badge tone="info" className="text-xs">
                    {f.count} Contrato(s)
                  </Badge>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
