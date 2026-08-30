"use client";

import { useEffect, useState } from "react";
import { Newspaper, Search, Calendar, Download, Link2 } from "lucide-react";
import { searchGazetteArticles, pdfUrlWithPage, GazetteArticle, TypesenseSearchResponse } from "@/lib/typesense-client";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { Card, CardContent } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { useToast } from "@/components/notifications/ToastProvider";
import {
  readSearchState,
  writeSearchState,
  dateInputToUnix,
  toCSV,
  downloadTextFile,
} from "@/lib/search-tools";

export default function DiarioSearchPage() {
  const initial = typeof window !== "undefined" ? readSearchState() : {};
  const { showToast } = useToast();
  const [query, setQuery] = useState(initial.q ?? "");
  const [editionType, setEditionType] = useState(initial.edition_type ?? "");
  const [dateFrom, setDateFrom] = useState(initial.from ?? "");
  const [dateTo, setDateTo] = useState(initial.to ?? "");
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [results, setResults] = useState<TypesenseSearchResponse<GazetteArticle>>({
    found: 0,
    page: 1,
    hits: [],
    facet_counts: [],
  });

  const searchArgs = () => ({
    query,
    editionType: editionType || undefined,
    dateFrom: dateInputToUnix(dateFrom),
    dateTo: dateInputToUnix(dateTo, true),
  });

  const handleSearch = async () => {
    setLoading(true);
    writeSearchState({ q: query, edition_type: editionType, from: dateFrom, to: dateTo });
    try {
      const res = await searchGazetteArticles({ ...searchArgs(), page: 1, perPage: 20 });
      setResults(res);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleExportCSV = async () => {
    setExporting(true);
    try {
      const res = await searchGazetteArticles({ ...searchArgs(), page: 1, perPage: 250 });
      const rows = res.hits.map((h) => {
        const d = h.document;
        return {
          data: d.publication_date ? new Date(d.publication_date * 1000).toLocaleDateString("pt-BR") : "",
          edicao: d.edition_number,
          tipo_edicao: d.edition_type,
          pagina: d.page_number,
          contratos: (d.contract_numbers ?? []).join(" | "),
          cnpjs: (d.cnpjs ?? []).join(" | "),
          citados: (d.officials_named ?? []).join(" | "),
          conteudo: d.content,
          pdf: pdfUrlWithPage(d.pdf_storage_url, d.page_number),
        };
      });
      const csv = toCSV(rows, [
        { key: "data", label: "Data" },
        { key: "edicao", label: "Edição" },
        { key: "tipo_edicao", label: "Tipo" },
        { key: "pagina", label: "Página" },
        { key: "contratos", label: "Contratos" },
        { key: "cnpjs", label: "CNPJs" },
        { key: "citados", label: "Citados" },
        { key: "conteudo", label: "Conteúdo" },
        { key: "pdf", label: "PDF Oficial" },
      ]);
      downloadTextFile(`diario-oficial-${new Date().toISOString().slice(0, 10)}.csv`, csv);
      showToast({ title: "CSV exportado", description: `${rows.length} registro(s).`, tone: "success" });
    } catch {
      showToast({ title: "Falha ao exportar CSV", tone: "danger" });
    } finally {
      setExporting(false);
    }
  };

  const copyPermalink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      showToast({ title: "Link copiado", description: "A busca atual está no seu clipboard.", tone: "info" });
    } catch {
      showToast({ title: "Não foi possível copiar o link", tone: "danger" });
    }
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- busca de dados no mount/na mudança de filtro; migração pra SWR (useApiQuery) é item à parte (audit-2026-08, item 4).
    handleSearch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editionType, dateFrom, dateTo]);

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-surface-border pb-4">
        <div>
          <p className="dateline">DIORONDON · Busca em texto integral</p>
          <h1 className="mt-2 text-2xl font-semibold text-foreground">Pesquisa no Diário Oficial</h1>
          <p className="mt-1 text-sm text-muted">
            Contratos, editais, leis, portarias e decretos da Administração Municipal, no texto
            completo de cada edição.
          </p>
        </div>
      </div>

      {/* Barra de Pesquisa Principal */}
      <div className="flex flex-col sm:flex-row gap-3 rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted pointer-events-none" />
          <Input
            placeholder="Digite o número do contrato, CNPJ, nome da empresa ou palavra-chave..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="pl-10 text-sm"
          />
        </div>

        <select
          value={editionType}
          onChange={(e) => setEditionType(e.target.value)}
          className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs font-medium text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value="">Todas as Edições</option>
          <option value="ORDINARIA">Ordinária</option>
          <option value="SUPLEMENTAR">Suplementar / Extra</option>
        </select>

        <input
          type="date"
          value={dateFrom}
          onChange={(e) => setDateFrom(e.target.value)}
          className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs text-foreground"
          aria-label="Data inicial"
        />
        <input
          type="date"
          value={dateTo}
          onChange={(e) => setDateTo(e.target.value)}
          className="rounded-md border border-surface-border bg-surface px-3 py-2 text-xs text-foreground"
          aria-label="Data final"
        />

        <Button onClick={handleSearch} disabled={loading} size="sm">
          {loading ? "Pesquisando..." : "Pesquisar"}
        </Button>
      </div>

      {/* Lista de Resultados */}
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
          <span>
            Encontrados <strong>{results.found}</strong> artigos/extratos
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={copyPermalink} className="text-xs inline-flex items-center gap-1.5">
              <Link2 size={13} />
              Copiar link
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={handleExportCSV}
              disabled={exporting || results.found === 0}
              className="text-xs inline-flex items-center gap-1.5"
            >
              <Download size={13} />
              {exporting ? "Exportando…" : "Exportar CSV"}
            </Button>
          </div>
        </div>

        {results.hits.length === 0 ? (
          <div className="rounded-xl border border-dashed border-surface-border bg-surface p-12 text-center text-muted">
            <Newspaper size={32} className="mx-auto text-muted/40 mb-3" />
            <p className="font-semibold text-foreground">Nenhum artigo encontrado para os termos informados.</p>
            <p className="text-xs mt-1">Tente buscar por termos mais genéricos ou pelo número de contrato/CNPJ.</p>
          </div>
        ) : (
          results.hits.map((hit, idx) => {
            const doc = hit.document;
            const highlightSnippet = hit.highlights?.find((h) => h.field === "content")?.snippet || doc.content;

            return (
              <Card key={doc.id || idx} className="hover:border-primary/50 transition-colors">
                <CardContent className="p-5 flex flex-col gap-3">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-surface-border/60 pb-3">
                    <div className="flex items-center gap-2">
                      <Badge tone={doc.edition_type === "ORDINARIA" ? "info" : "warning"}>
                        Edição {doc.edition_type || "ORDINÁRIA"} #{doc.edition_number}
                      </Badge>
                      <span className="text-xs text-muted font-mono">Página {doc.page_number}</span>
                    </div>

                    <div className="flex items-center gap-2 text-xs text-muted font-mono">
                      <Calendar size={12} />
                      <span>{new Date(doc.publication_date * 1000).toLocaleDateString("pt-BR")}</span>
                    </div>
                  </div>

                  {/* Metadados extraídos: Contratos e CNPJs */}
                  {(doc.contract_numbers?.length > 0 || doc.cnpjs?.length > 0) && (
                    <div className="flex flex-wrap items-center gap-2 text-xs">
                      {doc.contract_numbers?.map((c) => (
                        <span key={c} className="rounded bg-primary/10 px-2 py-0.5 font-mono text-[11px] font-semibold text-primary">
                          Contrato {c}
                        </span>
                      ))}
                      {doc.cnpjs?.map((cnpj) => (
                        <span key={cnpj} className="rounded bg-surface-border/60 px-2 py-0.5 font-mono text-[11px] text-muted">
                          CNPJ {cnpj}
                        </span>
                      ))}
                    </div>
                  )}

                  {/* Trecho com highlight — renderizado com segurança (o snippet
                      do Typesense ecoa texto de PDF ingerido; NÃO usar
                      dangerouslySetInnerHTML). */}
                  <div className="rounded-md bg-surface-hover/50 p-3 text-xs text-foreground/90 font-sans border-l-2 border-primary leading-relaxed">
                    <SafeHighlight snippet={highlightSnippet} />
                  </div>

                  {doc.pdf_storage_url && (
                    <div className="flex justify-end pt-1">
                      <a
                        href={pdfUrlWithPage(doc.pdf_storage_url, doc.page_number)}
                        target="_blank"
                        rel="noreferrer"
                        className="text-xs text-primary hover:underline font-medium inline-flex items-center gap-1"
                      >
                        {doc.page_number > 1
                          ? `Ver no PDF oficial (pág. ${doc.page_number}) →`
                          : "Ver PDF da Edição Oficial →"}
                      </a>
                    </div>
                  )}
                </CardContent>
              </Card>
            );
          })
        )}
      </div>
    </div>
  );
}

/** Renderiza um snippet de destaque do Typesense sem `dangerouslySetInnerHTML`:
 *  o snippet ecoa o `content` indexado (texto bruto de PDFs) apenas envolvendo
 *  o trecho casado em <mark> — o resto NÃO é escapado. Aqui todo o texto é
 *  tratado como texto puro (React escapa) e só o realce <mark> é reintroduzido
 *  como elemento real. */
function SafeHighlight({ snippet }: { snippet: string }) {
  // Divide o snippet mantendo cada trecho <mark>…</mark> como um item
  // próprio — assim cada parte se descreve sozinha (sem variável mutável
  // no closure do .map, que o React Compiler proíbe).
  const parts = snippet.split(/(<mark>[\s\S]*?<\/mark>)/g);
  return (
    <p className="line-clamp-4">
      {parts.map((part, i) => {
        if (!part) return null;
        const marked = part.match(/^<mark>([\s\S]*)<\/mark>$/);
        return marked ? (
          <mark key={i} className="rounded bg-primary/20 px-0.5 text-foreground">
            {marked[1]}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        );
      })}
    </p>
  );
}
