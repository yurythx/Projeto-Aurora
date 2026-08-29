// Utilitários compartilhados pelas telas de busca do DIORONDON
// (/pessoal e /diario): permalink (estado da busca na URL), filtro por
// período e exportação CSV do resultado corrente.

/** Lê os parâmetros da URL atual como objeto simples (client-side). */
export function readSearchState(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const out: Record<string, string> = {};
  new URLSearchParams(window.location.search).forEach((v, k) => {
    if (v) out[k] = v;
  });
  return out;
}

/** Reescreve a query string da URL sem recarregar a página (permalink).
 *  Chaves com valor vazio/undefined são removidas. */
export function writeSearchState(state: Record<string, string | number | undefined>): void {
  if (typeof window === "undefined") return;
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(state)) {
    if (v === undefined || v === null || v === "") continue;
    params.set(k, String(v));
  }
  const qs = params.toString();
  const url = qs ? `${window.location.pathname}?${qs}` : window.location.pathname;
  window.history.replaceState(null, "", url);
}

/** "YYYY-MM-DD" -> segundos Unix (início do dia, hora local). "" -> undefined. */
export function dateInputToUnix(value: string, endOfDay = false): number | undefined {
  if (!value) return undefined;
  // Interpreta o dia em UTC: publication_date é EditionDate.Unix() (data da
  // edição à meia-noite UTC). Usar horário local aqui deslocaria o limite em
  // algumas horas e poderia excluir a edição publicada exatamente na data.
  const [y, m, d] = value.split("-").map(Number);
  if (!y || !m || !d) return undefined;
  const ms = endOfDay
    ? Date.UTC(y, m - 1, d, 23, 59, 59)
    : Date.UTC(y, m - 1, d, 0, 0, 0);
  return Math.floor(ms / 1000);
}

/** Monta o fragmento filter_by do Typesense para um intervalo de
 *  publication_date (int64 unix). Devolve [] quando nenhum limite é dado. */
export function publicationDateFilter(fromUnix?: number, toUnix?: number): string[] {
  const parts: string[] = [];
  if (fromUnix != null) parts.push(`publication_date:>=${fromUnix}`);
  if (toUnix != null) parts.push(`publication_date:<=${toUnix}`);
  return parts;
}

function csvCell(v: unknown): string {
  const s = v == null ? "" : String(v);
  return /[",\n;]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

/** Gera um CSV (separador ';', cabeçalho pelas chaves de `columns`). */
export function toCSV<T extends Record<string, unknown>>(
  rows: T[],
  columns: { key: keyof T; label: string }[],
): string {
  const header = columns.map((c) => csvCell(c.label)).join(";");
  const body = rows
    .map((r) => columns.map((c) => csvCell(r[c.key])).join(";"))
    .join("\n");
  // BOM (U+FEFF) para o Excel abrir a acentuação corretamente.
  return `\uFEFF${header}\n${body}\n`;
}

/** Dispara o download de um arquivo texto no navegador (origem do app —
 *  não é o sandbox de artifact, então <a download> funciona). */
export function downloadTextFile(filename: string, content: string, mime = "text/csv;charset=utf-8"): void {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function unixToISODate(unixSeconds: number | undefined): string {
  if (!unixSeconds) return "";
  return new Date(unixSeconds * 1000).toISOString().slice(0, 10);
}
