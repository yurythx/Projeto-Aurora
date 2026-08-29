import { apiClient } from "./client";
import type { Contrato, KanbanResponse, PaginationMeta } from "@/types/api";

export async function fetchKanbanView(limit: number = 20): Promise<KanbanResponse> {
  const { data } = await apiClient.get<KanbanResponse>(`/v1/contratos/kanban?limit=${limit}`);
  if (!data) throw new Error("Failed to load kanban view");
  return data;
}

export async function fetchContratos(
  page: number = 1,
  pageSize: number = 20,
  status?: string,
  busca?: string
): Promise<{ data: Contrato[]; meta: PaginationMeta }> {
  const params = new URLSearchParams({ page: page.toString(), page_size: pageSize.toString() });
  if (status) params.append("status", status);
  if (busca) params.append("busca", busca);

  // O endpoint pode retornar meta, ou podemos criar manualmente a partir de {data, total, page, pages}
  // Vamos assumir que criamos o envolucro na resposta
  const res = await apiClient.get<{ data: Contrato[]; total: number; page: number; pages: number }>(
    `/v1/contratos?${params.toString()}`
  );
  if (!res.data) throw new Error("Failed to load contratos");
  return {
    data: res.data.data,
    meta: {
      page: res.data.page,
      page_size: pageSize,
      total_items: res.data.total,
      total_pages: res.data.pages,
    },
  };
}

export async function updateContratoStatus(id: string, status: string): Promise<void> {
  await apiClient.patch(`/v1/contratos/${id}/status`, { status });
}

// Detalhe completo de um contrato — inclui diario_refs (as publicações do
// Diário Oficial vinculadas, populadas pelo casador automático) e aditivos.
export async function getContratoDetail(id: string): Promise<Contrato> {
  const { data } = await apiClient.get<Contrato>(`/v1/contratos/${id}`);
  if (!data) throw new Error("Contrato não encontrado");
  return data;
}
