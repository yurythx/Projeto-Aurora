// Formatos compartilhados que espelham os DTOs do backend do Projeto-Nova.
// Mantidos manualmente em sincronia com docs/openapi.yaml — qualquer novo
// campo exposto pela API precisa ser refletido aqui para o frontend
// enxergá-lo com tipagem.

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  active: boolean;
  created_at: string;
  last_seen_at?: string;
}

export type IntegrationStatus = "unknown" | "online" | "offline" | "degraded" | "disabled";

export interface Integration {
  id: string;
  key: string;
  name: string;
  type: string;
  enabled: boolean;
  status: IntegrationStatus;
  last_check_at?: string;
  last_success_at?: string;
  last_error?: string;
}

export type JobStatus = "queued" | "processing" | "completed" | "failed" | "dead_letter";

export interface TestJobResponse {
  job_id: string;
  status: JobStatus;
}

export interface PaginationMeta {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
}

// GET /api/v1/admin/feature-flags (restrito a nova-admin) — ver docs/openapi.yaml.
export interface FeatureFlag {
  key: string;
  enabled: boolean;
  description?: string;
}



// MVP de monitoramento real do Diário Oficial via DJEN — o que o
// usuário quer acompanhar (OAB+UF, número de processo, ou texto livre;
// pelo menos um preenchido, validado no backend). Espelha
// transport.monitoredTermResponse.
export interface MonitoredTerm {
  id: string;
  label: string;
  oab_number?: string;
  oab_uf?: string;
  process_number?: string;
  free_text?: string;
  active: boolean;
  last_synced_at?: string;
  created_at: string;
}

// GET /api/v1/diario-oficial/health — mesmo shape que ScannerHealth
// (source no lugar de scanner): a checagem síncrona da fonte de dados
// configurada (DJEN hoje), pra uma tela mostrar "está respondendo?"
// antes do usuário estranhar por que nada de novo apareceu.
export interface SourceHealth {
  source: string;
  healthy: boolean;
  message?: string;
  checked_at: string;
}

// Uma publicação do DJEN que casou com um MonitoredTerm — espelha
// transport.matchedPublicationResponse.
export interface MatchedPublication {
  id: string;
  tribunal: string;
  orgao: string;
  tipo_comunicacao: string;
  texto: string;
  process_number: string;
  process_number_masked: string;
  availability_date: string;
  link?: string;
  monitored_term_id: string;
  monitored_term_label: string;
  matched_at: string;
}

export type HREventType = "EXONERACAO" | "NOMEACAO" | "MUDANCA_SETOR";

export interface HREvent {
  type: HREventType;
  servidor: string;
  servidor_cpf?: string;
  servidor_matricula?: string;
  secretaria?: string;
  cargo?: string;
  das_level?: string;
  portaria_number: string;
  edition_number: string;
  context_snippet: string;
  doc_url: string;
  publication_date: string;
}

export interface PublicContract {
  id: string;
  contract_number: string;
  contract_type: string;
  portaria_number: string;
  nomeacao_date: string;
  object: string;
  contractor: string;
  contractor_cnpj: string;
  value: string;
  fiscal_nome: string;
  fiscal_cpf: string;
  fiscal_matricula: string;
  suplente_nome: string;
  suplente_cpf: string;
  suplente_matricula: string;
  edition_number: string;
  publication_date: string;
  doc_url?: string;
  status: string;
}

// --- Módulo de Contratos (Novo) ---

export interface Contrato {
  id: string;
  numero: string;
  objeto: string;
  contratante: string;
  contratado: string;
  cnpj?: string;
  valor?: number;
  data_assinatura?: string;
  data_vigencia_inicio?: string;
  data_vigencia_fim?: string;
  status: string;
  status_label: string;
  diario_refs?: DiarioRef[];
  aditivos?: Aditivo[];
  created_at: string;
  updated_at: string;
}

export interface DiarioRef {
  edition_number: string;
  tipo_evento?: string;
  publicado_em?: string;
  contexto?: string;
  doc_url?: string;
}

export interface Aditivo {
  id: string;
  numero?: string;
  objeto?: string;
  valor_adicional?: number;
  data_assinatura?: string;
  created_at: string;
}

export interface KanbanColumn {
  status: string; // "1", "2", "3" (representando as Etapas)
  label: string;
  total: number;
  items: DemandResponse[]; // Modificado para usar Demanda em vez de Contrato genérico
}

export interface KanbanResponse {
  columns: KanbanColumn[];
}

export interface DocumentResponse {
  id: string;
  doc_type: string;
  file_path: string;
  file_name: string;
  uploaded_at: string;
  validade_ate?: string;
}

export interface DemandResponse {
  id: string;
  contrato_id: string;
  ano_mes: string;
  etapa: number;
  status_etapa: string;
  observacoes: string;
  etapa_started_at: string;
  documents?: DocumentResponse[];
  // Vamos injetar o numero do contrato no frontend para exibir no card
  contrato_numero?: string;
  contrato_objeto?: string;
  contratado?: string;
  contrato_valor?: number;
}
