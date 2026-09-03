"use client";

import { Accessibility, Eye } from "lucide-react";
import Link from "next/link";
import { useBranding } from "@/components/branding/BrandingContext";

// Barra e-MAG de Acessibilidade Governamental (Prefeitura Municipal de
// Rondonópolis) — faixa fixa de 40px no topo do shell. Reúne, num só lugar:
//  - Skip links (Alt+1 conteúdo, Alt+2 menu, Alt+3 busca, Alt+4 rodapé),
//    visíveis só ao receber foco por teclado;
//  - Identidade institucional (órgão + descrição do sistema);
//  - Link para a página dedicada /acessibilidade;
//  - Alternância de Alto Contraste (paleta e-MAG preto/amarelo);
//  - Redimensionamento de fonte A+ / A- / A (90%–130% do documento).
export function EMagAccessibilityBar({
  onOpenGlobalSearch,
}: {
  onOpenGlobalSearch?: () => void;
}) {
  const {
    branding,
    toggleHighContrast,
    increaseFontSize,
    decreaseFontSize,
    resetFontSize,
  } = useBranding();

  return (
    <div className="flex h-10 shrink-0 items-center justify-between bg-header-topbar px-3 sm:px-4 text-xs font-semibold text-white shadow-xs">
      {/* Skip Links e-MAG — invisíveis até receberem foco por teclado (Alt+1..Alt+4) */}
      <div className="sr-only focus-within:not-sr-only focus-within:absolute focus-within:z-50 focus-within:flex focus-within:bg-header-topbar focus-within:p-2 focus-within:gap-2">
        <a
          href="#conteudo"
          accessKey="1"
          className="rounded px-2 py-1 bg-white text-slate-900 font-bold hover:bg-amber-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-header-topbar focus-visible:ring-white"
          title="Ir para o conteúdo principal (Alt + 1)"
        >
          [Alt+1] Ir para Conteúdo
        </a>
        <a
          href="#menu"
          accessKey="2"
          className="rounded px-2 py-1 bg-white text-slate-900 font-bold hover:bg-amber-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-header-topbar focus-visible:ring-white"
          title="Ir para o menu de navegação (Alt + 2)"
        >
          [Alt+2] Ir para Menu
        </a>
        {/* Atalho de busca só quando há uma busca global para abrir. Sem isso o
            atalho ficava "morto" numa tela sem #busca (A-14). */}
        {onOpenGlobalSearch && (
          <button
            type="button"
            accessKey="3"
            onClick={onOpenGlobalSearch}
            className="rounded px-2 py-1 bg-white text-slate-900 font-bold hover:bg-amber-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-header-topbar focus-visible:ring-white"
            title="Abrir busca global (Alt + 3)"
          >
            [Alt+3] Ir para Busca
          </button>
        )}
        <a
          href="#rodape"
          accessKey="4"
          className="rounded px-2 py-1 bg-white text-slate-900 font-bold hover:bg-amber-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-header-topbar focus-visible:ring-white"
          title="Ir para o rodapé (Alt + 4)"
        >
          [Alt+4] Ir para Rodapé
        </a>
      </div>

      {/* Lado Esquerdo: Identidade Institucional (min-w-0 para o truncate
          funcionar em telas de 320px — D-06) */}
      <div className="flex min-w-0 items-center gap-2 overflow-hidden text-white/90 sm:gap-3">
        <span className="truncate text-[11px] font-semibold uppercase tracking-wider text-white sm:text-xs">
          {branding.orgName}
        </span>
        <span className="hidden md:inline text-white/40">•</span>
        <span className="hidden md:inline truncate text-white/80 font-normal text-[11px] sm:text-xs">
          {branding.appDescription}
        </span>
      </div>

      {/* Lado Direito: Ferramentas e-MAG (Acessibilidade, Contraste, Fontes A+/A-/A) */}
      <div className="flex items-center gap-2 sm:gap-3 shrink-0">
        <Link
          href="/acessibilidade"
          aria-label="Página de Acessibilidade Oficial"
          className="flex items-center gap-1.5 hover:bg-white/15 transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-white rounded-md px-2 py-1 text-white"
          title="Página de Acessibilidade — Instruções e Atalhos"
        >
          <Accessibility size={15} className="text-sky-300" />
          <span className="hidden sm:inline">Acessibilidade</span>
        </Link>

        <span className="text-white/30 hidden sm:inline">•</span>

        <button
          type="button"
          onClick={toggleHighContrast}
          aria-label="Alternar Modo Alto Contraste (Acessibilidade e-MAG)"
          className="flex items-center gap-1.5 hover:bg-white/15 transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-white rounded-md px-2 py-1 text-white"
          title="Alto Contraste"
        >
          <Eye size={14} className={branding.highContrast ? "text-amber-300" : ""} />
          <span className="hidden sm:inline">
            {branding.highContrast ? "Alto Contraste: Ativo" : "Alto Contraste"}
          </span>
        </button>

        <span className="text-white/30">•</span>

        {/* Redimensionamento de Fonte A+ / A- / A */}
        <div className="flex items-center gap-0.5 font-bold text-xs bg-white/15 rounded-md p-0.5 border border-white/20">
          <button
            type="button"
            onClick={increaseFontSize}
            aria-label="Aumentar Fonte (A+)"
            className="hover:bg-white/25 focus:outline-none focus-visible:ring-2 focus-visible:ring-white rounded px-2 py-0.5 transition-colors text-white"
            title="Aumentar Fonte (A+)"
          >
            A+
          </button>
          <button
            type="button"
            onClick={decreaseFontSize}
            aria-label="Diminuir Fonte (A-)"
            className="hover:bg-white/25 focus:outline-none focus-visible:ring-2 focus-visible:ring-white rounded px-2 py-0.5 transition-colors text-white"
            title="Diminuir Fonte (A-)"
          >
            A-
          </button>
          <button
            type="button"
            onClick={resetFontSize}
            aria-label="Tamanho de Fonte Normal (A)"
            className="hover:bg-white/25 focus:outline-none focus-visible:ring-2 focus-visible:ring-white rounded px-2 py-0.5 transition-colors text-white"
            title="Restaurar Tamanho Normal de Fonte (A)"
          >
            A
          </button>
        </div>
      </div>
    </div>
  );
}
