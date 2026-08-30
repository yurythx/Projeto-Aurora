import { SectionTabs } from "@/components/layout/SectionTabs";
import { DIARIO_TABS } from "@/lib/nav/sectionTabs";
import { EtlPipelinePanel } from "@/components/diario-oficial/EtlPipelinePanel";
import { MatchedPublicationsFeed } from "@/components/diario-oficial/MatchedPublicationsFeed";
import { MonitoredTermsPanel } from "@/components/diario-oficial/MonitoredTermsPanel";
import { RawApiFeedPanel } from "@/components/diario-oficial/RawApiFeedPanel";
import { RondonopolisContractsFeed } from "@/components/diario-oficial/RondonopolisContractsFeed";
import { RondonopolisHREventsFeed } from "@/components/diario-oficial/RondonopolisHREventsFeed";
import { SourceHealthPanel } from "@/components/diario-oficial/SourceHealthPanel";
import { Section } from "@/components/ui/Section";

import { ExecutiveAnalyticsPanel } from "@/components/diario-oficial/ExecutiveAnalyticsPanel";

export default function DiarioOficialPage() {
  return (
    <div className="flex flex-col gap-8">
      <div>
        <p className="dateline">DIORONDON-E</p>
        <h1 className="mt-2 text-2xl font-semibold">Diário Oficial de Rondonópolis</h1>
        <p className="mt-1 text-sm text-muted">
          Portal do motor DIORONDON: saúde da fonte, pipeline de ingestão, retorno bruto da API e
          feeds de atos de pessoal e contratos públicos.
        </p>
      </div>

      <SectionTabs ariaLabel="Diário Oficial" tabs={DIARIO_TABS} />

      <SourceHealthPanel />

      {/* Painel Analítico & Indicadores Executivos */}
      <Section
        title="Painel Analítico de Indicadores Executivos & Métricas"
        description="Resumo de atos de pessoal, montante financeiro sob fiscalização e cargos comissionados DAS em Rondonópolis-MT."
        collapsible={true}
        defaultExpanded={true}
      >
        <ExecutiveAnalyticsPanel />
      </Section>

      {/* Painel de Controle de Ingestão ETL & Vigia */}
      <Section
        title="Painel de Controle de Pipeline ETL & Ingestão (Watcher)"
        description="Acompanhamento em tempo real das edições capturadas, status de processamento do Worker Pool e contagem de achados indexados no PostgreSQL."
        collapsible={true}
        defaultExpanded={true}
      >
        <EtlPipelinePanel />
      </Section>

      {/* Seção Transparente de Retorno Completo da API */}
      <Section
        title="Retorno Bruto da API & Auditoria de Edições (DIORONDON-E)"
        description="Lista completa de tudo o que a API oficial de Rondonópolis retorna, sem filtros ocultos, com inspeção de payload JSON e PDF oficial."
        collapsible={true}
        defaultExpanded={true}
      >
        <RawApiFeedPanel />
      </Section>

      {/* Seção 1: Atos de Pessoal (Exoneração, Contratação, Realocação) + DAS + Legenda */}
      <Section
        title="Atos de Pessoal (Exoneração, Contratação e Realocação)"
        description="Busca ultrarrápida de exonerações, nomeações/contratações e relotações de servidores, com níveis DAS e tabela de média salarial."
        collapsible={true}
        defaultExpanded={true}
      >
        <RondonopolisHREventsFeed />
      </Section>

      {/* Seção 2: Nova Consulta de Contratos Novos e Existentes */}
      <Section
        title="Consulta de Contratos Novos & Existentes"
        description="Pesquisa de contratos por Nome, CPF ou Matrícula. Traz tipo de contrato, portaria, data de nomeação, empresa, CNPJ, fiscal titular e suplente do contrato."
        collapsible={true}
        defaultExpanded={true}
      >
        <RondonopolisContractsFeed />
      </Section>

      {/* Seção 3: Termos Monitorados */}
      <Section
        title="Termos Monitorados"
        description="Palavras-chave e termos acompanhados no Diário Oficial de Rondonópolis."
        collapsible={true}
        defaultExpanded={false}
      >
        <MonitoredTermsPanel />
      </Section>

      {/* Seção 4: Publicações Recentes */}
      <Section
        title="Publicações Recentes"
        description="Publicações capturadas do Diário Oficial da cidade com filtros por termo e tipo."
        collapsible={true}
        defaultExpanded={true}
      >
        <MatchedPublicationsFeed />
      </Section>
    </div>
  );
}
