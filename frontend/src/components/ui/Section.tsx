"use client";

import { useState, type ReactNode } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

export function Section({
  title,
  description,
  action,
  collapsible = false,
  defaultExpanded = true,
  children,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  collapsible?: boolean;
  defaultExpanded?: boolean;
  children: ReactNode;
}) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <section className="rounded-xl border border-surface-border bg-surface shadow-sm transition-all duration-200">
      <div
        className={`flex flex-wrap items-center justify-between gap-3 border-b border-surface-border px-5 py-4 ${
          collapsible ? "cursor-pointer hover:bg-surface-border/10" : ""
        }`}
        onClick={(e) => {
          // Não fecha se clicar diretamente em um botão de ação
          if (collapsible && !(e.target as HTMLElement).closest("button, a, select, input")) {
            setIsExpanded(!isExpanded);
          }
        }}
      >
        <div className="flex-1 select-none">
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-foreground">{title}</h2>
            {collapsible && (
              <span className="text-xs text-muted font-medium px-2 py-0.5 rounded bg-surface-border/50">
                {isExpanded ? "Expandido" : "Minimizado (Clique para abrir)"}
              </span>
            )}
          </div>
          {description && <p className="mt-0.5 text-sm text-muted">{description}</p>}
        </div>
        <div className="flex items-center gap-2">
          {action}
          {collapsible && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setIsExpanded(!isExpanded);
              }}
              className="p-1.5 rounded-md hover:bg-surface-border/50 text-muted hover:text-foreground transition-colors"
              aria-label={isExpanded ? "Minimizar seção" : "Expandir seção"}
            >
              {isExpanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
            </button>
          )}
        </div>
      </div>
      {(!collapsible || isExpanded) && <div className="p-5">{children}</div>}
    </section>
  );
}
