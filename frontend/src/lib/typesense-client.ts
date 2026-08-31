import { safeHttpUrl } from "@/lib/search-tools";

export interface TypesenseSearchResponse<T> {
  found: number;
  page: number;
  hits: Array<{
    document: T;
    highlights: Array<{
      field: string;
      snippet: string;
      snippets?: string[];
    }>;
  }>;
  facet_counts?: Array<{
    field_name: string;
    counts: Array<{
      value: string;
      count: number;
    }>;
  }>;
}

const TYPESENSE_HOST = process.env.NEXT_PUBLIC_TYPESENSE_URL || "http://localhost:8109";


function emptyTypesenseResponse<T>(): TypesenseSearchResponse<T> {
  return { found: 0, page: 1, hits: [], facet_counts: [] };
}

export function pdfUrlWithPage(url: string | undefined, page: number | undefined): string {
  const safe = safeHttpUrl(url);
  if (!safe) return "";
  const p = Number(page) || 1;
  if (p > 1 && !safe.includes("#page=")) return `${safe}#page=${p}`;
  return safe;
}

export async function getTypesenseHealth(): Promise<{ healthy: boolean; message: string; docCount?: number }> {
  try {
    const res = await fetch(`${TYPESENSE_HOST}/health`, { method: "GET" });
    if (!res.ok) {
      return { healthy: false, message: `Typesense retornou status HTTP ${res.status}` };
    }
    return { healthy: true, message: "Motor de busca Typesense operacional", docCount: 0 };
  } catch (err) {
    return {
      healthy: false,
      message: err instanceof Error ? err.message : "Falha na conexão com o Typesense",
    };
  }
}
