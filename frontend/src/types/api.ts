// Formatos compartilhados que espelham os DTOs do backend do Projeto Aurora.
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

// GET /api/v1/admin/feature-flags (restrito a aurora-admin) — ver docs/openapi.yaml.
export interface FeatureFlag {
  key: string;
  enabled: boolean;
  description?: string;
}

// GET/PUT /api/v1/admin/keycloak (restrito a aurora-admin) — ver docs/openapi.yaml.
// "source" indica de onde vieram os valores: "database" (já salvo pelo
// menu Configurações > Keycloak — o que está em uso agora), "environment"
// (nunca foi salvo por esta tela; vem de variável de ambiente do
// processo) ou "unset". Nunca inclui um client secret em texto plano.
export interface KeycloakSettingsStatus {
  source: "database" | "environment" | "unset";
  issuer_url: string;
  realm: string;
  client_id: string;
  client_secret_set: boolean;
  audience: string;
  frontend_client_id: string;
  frontend_client_secret_set: boolean;
  updated_at?: string;
  updated_by?: string;
}

// POST /api/v1/admin/keycloak/test — resultado de um teste de conexão,
// nunca persiste nada.
export interface KeycloakTestResult {
  status: "ok" | "warning" | "failed";
  discovery_ok: boolean;
  discovery_message: string;
  credentials_checked: boolean;
  credentials_ok: boolean;
  credentials_message: string;
}
