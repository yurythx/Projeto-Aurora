import { apiClient } from "./client";
import type { AuditLogRow } from "@/types/api";

export async function getDemandHistory(id: string): Promise<AuditLogRow[]> {
  const { data } = await apiClient.get<AuditLogRow[]>(`/api/v1/demands/${id}/history`);
  return Array.isArray(data) ? data : [];
}

export async function updateDemandEtapa(id: string, targetEtapa: number): Promise<void> {
  await apiClient.patch(`/api/v1/demands/${id}/etapa`, { target_etapa: targetEtapa });
}

export async function createDemand(contratoId: string, anoMes: string, observacoes?: string): Promise<any> {
  const { data } = await apiClient.post(`/api/v1/demands`, {
    contrato_id: contratoId,
    ano_mes: anoMes,
    observacoes: observacoes || "",
  });
  return data;
}

export async function addDemandDocument(id: string, docType: string, file: File): Promise<void> {
  // 1. Pede ao backend uma URL pré-assinada (Presigned URL)
  const { data } = await apiClient.get<{upload_url: string, file_path: string}>(
    `/api/v1/demands/${id}/documents/upload-url?doc_type=${encodeURIComponent(docType)}&file_name=${encodeURIComponent(file.name)}`
  );
  const { upload_url, file_path } = data;

  // 2. Faz o upload diretamente pro MinIO usando a URL
  const uploadRes = await fetch(upload_url, {
    method: "PUT",
    body: file,
    headers: {
      "Content-Type": file.type || "application/octet-stream",
    },
  });

  if (!uploadRes.ok) {
    throw new Error(`Falha no upload para o MinIO: ${uploadRes.statusText}`);
  }

  // 3. Confirma pro backend que o arquivo subiu com sucesso
  await apiClient.post(`/api/v1/demands/${id}/documents`, { 
    doc_type: docType, 
    file_path: file_path,
    file_name: file.name
  });
}
