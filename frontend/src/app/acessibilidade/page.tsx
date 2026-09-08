import type { Metadata } from "next";
import Link from "next/link";
import { Accessibility, Eye, Type, ExternalLink, Keyboard, FileText } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";

export const metadata: Metadata = {
  title: "Acessibilidade | Prefeitura de Rondonópolis",
  description:
    "Declaração e instruções de acessibilidade e-MAG do portal oficial e sistemas da Prefeitura Municipal de Rondonópolis.",
};

export default function AccessibilityPage() {
  return (
    <PublicShell>
      <div className="mx-auto max-w-5xl px-4 py-8 flex flex-col gap-8">
        {/* Breadcrumb */}
        <nav
          aria-label="Navegação estrutural (Breadcrumb)"
          className="text-xs text-muted flex items-center gap-2"
        >
          <Link href="/" className="hover:text-primary transition-colors">
            Início
          </Link>
          <span>/</span>
          <span className="font-semibold text-foreground">Acessibilidade</span>
        </nav>

        {/* Cabeçalho Principal */}
        <div className="flex flex-col gap-3 border-b border-surface-border pb-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-white shadow-sm">
              <Accessibility size={24} />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-foreground">Acessibilidade</h1>
              <p className="text-sm text-muted">
                Portal Oficial da Prefeitura Municipal de Rondonópolis — Conformidade com as normas
                e-MAG (Modelo de Acessibilidade do Governo Federal).
              </p>
            </div>
          </div>
        </div>

        {/* Conteúdo Principal de Acessibilidade */}
        <div className="grid gap-6 md:grid-cols-2">
          {/* Card 1: Atalhos de Teclado (e-MAG) */}
          <section className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
            <div className="flex items-center gap-2 text-primary font-semibold text-lg border-b border-surface-border pb-3">
              <Keyboard size={20} />
              <h2>Atalhos de Teclado (e-MAG)</h2>
            </div>
            <p className="text-xs text-muted leading-relaxed">
              Os padrões de atalhos válidos em qualquer página do site e sistema são:
            </p>
            <ul className="flex flex-col gap-2.5 text-xs text-foreground font-mono">
              <li className="flex items-center gap-2 rounded-lg bg-surface-hover p-2 border border-surface-border">
                <span className="rounded bg-primary px-2 py-0.5 text-white font-bold">Alt + 1</span>
                <span>Início do conteúdo principal da página (`#conteudo`)</span>
              </li>
              <li className="flex items-center gap-2 rounded-lg bg-surface-hover p-2 border border-surface-border">
                <span className="rounded bg-primary px-2 py-0.5 text-white font-bold">Alt + 2</span>
                <span>Início do menu principal (`#menu`)</span>
              </li>
              <li className="flex items-center gap-2 rounded-lg bg-surface-hover p-2 border border-surface-border">
                <span className="rounded bg-primary px-2 py-0.5 text-white font-bold">Alt + 3</span>
                <span>Busca interna (`#busca`)</span>
              </li>
              <li className="flex items-center gap-2 rounded-lg bg-surface-hover p-2 border border-surface-border">
                <span className="rounded bg-primary px-2 py-0.5 text-white font-bold">Alt + 4</span>
                <span>Início do rodapé (`#rodape`)</span>
              </li>
            </ul>

            <div className="mt-2 rounded-lg border border-warning/30 bg-warning/10 p-3 text-[11px] text-foreground">
              <strong>Particularidades por Navegador:</strong>
              <ul className="mt-1 list-disc list-inside flex flex-col gap-1 opacity-90 font-sans">
                <li>
                  No Firefox (Windows/Linux): use <code>Alt + Shift + Número</code>
                </li>
                <li>
                  No Firefox (macOS): use <code>Ctrl + Alt + Número</code>
                </li>
                <li>
                  No Opera: use <code>Shift + Escape + Número</code>
                </li>
              </ul>
            </div>
          </section>

          {/* Card 2: Ferramentas de Contraste e Redimensionamento de Fonte */}
          <section className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
            <div className="flex items-center gap-2 text-primary font-semibold text-lg border-b border-surface-border pb-3">
              <Type size={20} />
              <h2>Ferramentas de Contraste e Fonte</h2>
            </div>
            <div className="flex flex-col gap-4 text-xs text-muted leading-relaxed">
              <div className="flex items-start gap-3">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-warning/20 text-warning">
                  <Eye size={18} />
                </div>
                <div>
                  <h3 className="font-semibold text-foreground text-sm">Alto Contraste</h3>
                  <p>
                    É possível alterar o contraste de qualquer página através da opção{" "}
                    <strong>&quot;Alto Contraste&quot;</strong> na barra superior. O layout mudará
                    para o padrão e-MAG com fundo preto e acentos amarelos (taxa de contraste 19:1
                    AAA).
                  </p>
                </div>
              </div>

              <div className="flex items-start gap-3">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Type size={18} />
                </div>
                <div>
                  <h3 className="font-semibold text-foreground text-sm">
                    Redimensionamento de Fonte (A+ / A-)
                  </h3>
                  <p>
                    Na barra superior de acessibilidade, utilize os botões <strong>A+</strong> para
                    aumentar o tamanho do texto do documento em até 130% e <strong>A-</strong> para
                    reduzir. O botão <strong>A</strong> restaura o tamanho original.
                  </p>
                </div>
              </div>
            </div>
          </section>

          {/* Card 3: VLibras */}
          <section className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
            <div className="flex items-center gap-2 text-primary font-semibold text-lg border-b border-surface-border pb-3">
              <Accessibility size={20} />
              <h2>VLibras (Língua Brasileira de Sinais)</h2>
            </div>
            <p className="text-xs text-muted leading-relaxed">
              Através do ícone na barra superior ou do botão abaixo, é possível utilizar a suíte do{" "}
              <strong>VLibras</strong> para traduzir todo o conteúdo do site
              para a Língua Brasileira de Sinais - LIBRAS.
            </p>
            <div className="mt-auto pt-2">
              <a
                href="http://www.vlibras.gov.br/"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white shadow hover:bg-primary-hover transition-colors"
              >
                <span>Acessar Portal do VLibras</span>
                <ExternalLink size={14} />
              </a>
            </div>
          </section>

          {/* Card 4: Leis e Decretos sobre Acessibilidade */}
          <section className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
            <div className="flex items-center gap-2 text-primary font-semibold text-lg border-b border-surface-border pb-3">
              <FileText size={20} />
              <h2>Legislação sobre Acessibilidade</h2>
            </div>
            <p className="text-xs text-muted leading-relaxed">
              O Portal da Prefeitura Municipal de Rondonópolis e a plataforma Projeto Aurora cumprem
              rigorosamente os seguintes normativos legais:
            </p>
            <ul className="flex flex-col gap-3 text-xs">
              <li className="flex flex-col gap-1 border-l-2 border-primary pl-3">
                <a
                  href="http://www.planalto.gov.br/ccivil_03/_ato2011-2014/2012/Decreto/D7724.htm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold text-primary hover:underline inline-flex items-center gap-1"
                >
                  <span>Decreto nº 7.724, de 16 de Maio de 2012</span>
                  <ExternalLink size={12} />
                </a>
                <span className="text-muted text-[11px]">
                  Regulamenta a Lei Federal nº 12.527 (Lei de Acesso à Informação — LAI).
                </span>
              </li>
              <li className="flex flex-col gap-1 border-l-2 border-primary pl-3">
                <a
                  href="http://www.planalto.gov.br/ccivil_03/_Ato2004-2006/2004/Decreto/D5296.htm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold text-primary hover:underline inline-flex items-center gap-1"
                >
                  <span>Decreto nº 5.296, de 02 de Dezembro de 2004</span>
                  <ExternalLink size={12} />
                </a>
                <span className="text-muted text-[11px]">
                  Regulamenta as Leis nº 10.048/2000 e nº 10.098/2000 sobre Acessibilidade.
                </span>
              </li>
            </ul>
          </section>
        </div>
      </div>
    </PublicShell>
  );
}
