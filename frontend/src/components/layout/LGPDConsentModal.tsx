"use client";

import { useEffect, useState } from "react";
import { ShieldCheck, Lock, FileText, CheckCircle2 } from "lucide-react";
import Link from "next/link";
import { useSession } from "next-auth/react";

import { Button } from "@/components/ui/Button";

const CURRENT_TERM_VERSION = "v1.0.0-2026";

// deviceId: identificador opaco e aleatório do NAVEGADOR (não derivado de
// nenhum dado pessoal), persistido em localStorage. Serve só para o
// registro do consentimento anônimo (gap G-11) não duplicar e para o
// mesmo visitante, ao autenticar depois, ter o aceite reconhecível.
function getOrCreateDeviceId(): string {
  try {
    let id = localStorage.getItem("aurora_device_id");
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem("aurora_device_id", id);
    }
    return id;
  } catch {
    return crypto.randomUUID();
  }
}

export function LGPDConsentModal() {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const { status } = useSession();

  useEffect(() => {
    try {
      const consent = localStorage.getItem("aurora_lgpd_consent");
      if (consent === CURRENT_TERM_VERSION) return;
    } catch {
      // localStorage indisponível — mostra o modal mesmo assim
    }
    // 1.2s, não 0ms: na 1ª visita este backdrop cobre a tela bem na hora
    // em que um toast de login/logout pode estar chamando atenção — o
    // atraso deixa o toast aparecer antes de o modal disputar o foco.
    const timer = setTimeout(() => setIsOpen(true), 1200);
    return () => clearTimeout(timer);
  }, []);

  const handleAccept = async () => {
    setLoading(true);
    try {
      if (status === "authenticated") {
        await fetch("/api/backend/api/v1/lgpd/accept", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ term_version: CURRENT_TERM_VERSION }),
        });
      } else {
        // Gap G-11: visitante não autenticado — registro anônimo (sem PII)
        // no backend, não só no localStorage.
        await fetch("/api/backend/api/v1/lgpd/accept-anon", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            device_hash: getOrCreateDeviceId(),
            term_version: CURRENT_TERM_VERSION,
          }),
        });
      }
    } catch {
      // Fallback gracioso se a rota estiver indisponível
    } finally {
      try {
        localStorage.setItem("aurora_lgpd_consent", CURRENT_TERM_VERSION);
      } catch {
        /* sem localStorage — o modal reaparece na próxima visita */
      }
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
              Termos de Privacidade &amp; Proteção de Dados (LGPD)
            </h2>
            <p className="text-xs text-muted">Lei Federal Nº 13.709/2018</p>
          </div>
        </div>

        <div id="lgpd-description" className="flex flex-col gap-3 text-xs text-muted leading-relaxed">
          <p>
            Para garantir a transparência e a segurança da sua navegação, a{" "}
            <strong>Prefeitura Municipal de Rondonópolis</strong> utiliza a infraestrutura do{" "}
            <strong>Projeto Aurora</strong> com observância à LGPD.
          </p>
          <div className="flex flex-col gap-2 rounded-xl bg-surface-hover p-3 border border-surface-border">
            <div className="flex items-start gap-2">
              <Lock size={14} className="mt-0.5 text-success shrink-0" aria-hidden="true" />
              <span>
                <strong>Trilha imutável:</strong> seu aceite é registrado em auditoria protegida para
                garantir a autenticidade do consentimento.
              </span>
            </div>
            <div className="flex items-start gap-2">
              <FileText size={14} className="mt-0.5 text-primary shrink-0" aria-hidden="true" />
              <span>
                <strong>Mascaramento de dados:</strong> dados pessoais (CPF, e-mail, telefone) são
                higienizados e nunca aparecem em texto puro nos registros.
              </span>
            </div>
          </div>
          <p>
            Leia a{" "}
            <Link href="/privacidade" className="text-primary underline hover:no-underline">
              Política de Privacidade completa
            </Link>{" "}
            e os{" "}
            <Link href="/padroes" className="text-primary underline hover:no-underline">
              padrões e parâmetros aplicados
            </Link>
            .
          </p>
        </div>

        <div className="flex flex-wrap items-center justify-end gap-3 pt-2 border-t border-surface-border">
          <Button size="md" onClick={handleAccept} loading={loading}>
            <CheckCircle2 size={16} aria-hidden="true" />
            Concordar e Continuar
          </Button>
        </div>
      </div>
    </div>
  );
}
