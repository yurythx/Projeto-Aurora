"use client";

import { useEffect, useRef, type ReactNode } from "react";

type DialogSize = "sm" | "md" | "lg" | "xl";

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  /** Largura máxima do modal. `md` (padrão) serve pra confirmações; `lg`/`xl`
   *  pra modais com formulário, tabela ou lista. */
  size?: DialogSize;
  /** Rodapé fixo (não rola com o corpo) — ex.: botões de ação. O modal já
   *  põe o fio de separação e o espaçamento; o alinhamento fica com quem
   *  chama (ex.: `ml-auto` num botão pra jogá-lo à direita). */
  footer?: ReactNode;
  children?: ReactNode;
}

const sizeClass: Record<DialogSize, string> = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-2xl",
  xl: "max-w-3xl",
};

/**
 * Construído sobre o elemento nativo <dialog>: captura de foco,
 * fechar-com-Escape e o backdrop vêm todos do navegador em vez de serem
 * reimplementados — menos bugs de acessibilidade para errar.
 *
 * Dimensões (§ correção 2026-08): antes o <dialog> não tinha largura e o
 * conteúdo era espremido num wrapper interno de `max-w-md` fixo, então
 * qualquer modal com um <select> + botão lado a lado (ContratoModal)
 * estourava. Agora a largura é `min(100vw-2rem, size)` e o corpo rola
 * dentro de uma altura máxima em vez de vazar pra fora da tela.
 */
export function Dialog({
  open,
  onClose,
  title,
  description,
  size = "md",
  footer,
  children,
}: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      onClose={onClose}
      onCancel={onClose}
      aria-labelledby="dialog-title"
      aria-describedby={description ? "dialog-description" : undefined}
      className={`m-auto w-[calc(100vw-2rem)] ${sizeClass[size]} max-h-[calc(100dvh-4rem)]
        overflow-hidden rounded-xl border border-surface-border bg-surface p-0 text-foreground
        shadow-lg backdrop:bg-black/40`}
    >
      <div className="flex max-h-[calc(100dvh-4rem)] flex-col p-5">
        <h2 id="dialog-title" className="shrink-0 text-base font-semibold">
          {title}
        </h2>
        {description && (
          <p id="dialog-description" className="mt-1 shrink-0 text-sm text-muted">
            {description}
          </p>
        )}
        <div className="mt-4 min-h-0 flex-1 overflow-y-auto">{children}</div>
        {footer && (
          <div className="mt-4 flex shrink-0 flex-wrap items-center gap-2 border-t border-surface-border pt-4">
            {footer}
          </div>
        )}
      </div>
    </dialog>
  );
}
