"use client";

import { useEffect, useState } from "react";
import { ShieldCheck, Lock, FileText, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/Button";

const CURRENT_TERM_VERSION = "v1.0.0-2026";

export function LGPDConsentModal() {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);

  useEffect(() => {
    const consent = localStorage.getItem("aurora_lgpd_consent");
    if (consent !== CURRENT_TERM_VERSION) {
      // 1.2s, não 0ms: achado de revisão — na 1ª visita (consentimento
      // ainda não aceito), este backdrop cobre a tela inteira bem na hora
      // em que um toast de login/logout (AuthFlashToast) pode estar
      // tentando chamar atenção — abrir instantâneo arriscava a pessoa
      // nunca notar o toast, só o modal tomando conta da tela. Um atraso
      // pequeno dá tempo do toast já estar visível/registrado antes do
      // modal disputar atenção; a rolagem/foco só passam a ficar presos
      // no modal depois desse respiro, nunca de forma abrupta no 1º
      // pintar da página.
      const timer = setTimeout(() => setIsOpen(true), 1200);
      return () => clearTimeout(timer);
    }
  }, []);

  const handleAccept = async () => {
    setLoading(true);
    try {
      // Registra a aceitação na API
      await fetch("/api/backend/api/v1/lgpd/accept", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ term_version: CURRENT_TERM_VERSION }),
      });
    } catch {
      // Fallback gracioso se a rota for acessada offline
    } finally {
      localStorage.setItem("aurora_lgpd_consent", CURRENT_TERM_VERSION);
      setLoading(false);
      setIsOpen(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="lgpd-title"
      aria-describedby="lgpd-description"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4 animate-in fade-in duration-200"
    >
      <div className="flex w-full max-w-lg flex-col gap-5 rounded-2xl border border-surface-border bg-surface p-6 shadow-2xl">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <ShieldCheck size={24} aria-hidden="true" />
          </span>
          <div>
            <h2 id="lgpd-title" className="text-lg font-bold text-foreground">
              Termos de Privacidade & Proteção de Dados (LGPD)
            </h2>
            <p className="text-xs text-muted">Lei Federal Nº 13.709/2018</p>
          </div>
        </div>

        <div id="lgpd-description" className="flex flex-col gap-3 text-xs text-muted leading-relaxed">
          <p>
            Para garantir a transparência e segurança da sua navegação, a **Prefeitura Municipal de Rondonópolis** utiliza a infraestrutura do **Projeto Aurora** com total observância à LGPD.
          </p>
          <div className="flex flex-col gap-2 rounded-xl bg-surface-hover p-3 border border-surface-border">
            <div className="flex items-start gap-2">
              <Lock size={14} className="mt-0.5 text-success shrink-0" aria-hidden="true" />
              <span>
                <strong>Trilha Imutável:</strong> Seu aceite é registrado em auditoria protegida para garantir a autenticidade do consentimento.
              </span>
            </div>
            <div className="flex items-start gap-2">
              <FileText size={14} className="mt-0.5 text-primary shrink-0" aria-hidden="true" />
              <span>
                <strong>Mascaramento PII:</strong> Dados pessoais sensíveis (CPF, e-mail e telefone) são higienizados e nunca expostos em logs públicos.
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-end gap-3 pt-2 border-t border-surface-border">
          <Button
            size="md"
            onClick={handleAccept}
            loading={loading}
          >
            <CheckCircle2 size={16} aria-hidden="true" />
            Concordar e Continuar
          </Button>
        </div>
      </div>
    </div>
  );
}
