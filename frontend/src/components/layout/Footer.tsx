"use client";

import { useBranding } from "@/components/branding/BrandingContext";
import { Mail, Phone, Clock, ShieldCheck, ExternalLink } from "lucide-react";
import Link from "next/link";

export function Footer() {
  const { branding } = useBranding();
  const currentYear = new Date().getFullYear();

  return (
    <footer className="mt-auto border-t border-surface-border bg-surface text-xs text-muted">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Coluna 1: Nome da Aplicação e Orgão */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 font-bold text-foreground text-sm">
              <ShieldCheck className="h-5 w-5 text-primary" />
              <span>{branding.appName}</span>
            </div>
            <p className="text-muted leading-relaxed">{branding.appDescription}</p>
            <p className="text-[11px] font-semibold text-primary">{branding.orgName}</p>
          </div>

          {/* Coluna 2: Canais de Suporte e Atendimento */}
          <div className="space-y-2.5">
            <h4 className="font-bold text-foreground text-xs uppercase tracking-wider">Suporte & Atendimento</h4>
            <ul className="space-y-2">
              <li className="flex items-center gap-2">
                <Mail className="h-4 w-4 text-primary shrink-0" />
                <a href={`mailto:${branding.supportEmail}`} className="hover:text-primary transition-colors">
                  {branding.supportEmail}
                </a>
              </li>
              <li className="flex items-center gap-2">
                <Phone className="h-4 w-4 text-primary shrink-0" />
                <span>{branding.supportPhone}</span>
              </li>
              <li className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-primary shrink-0" />
                <span>{branding.supportHours}</span>
              </li>
            </ul>
          </div>

          {/* Coluna 3: Conformidade & Links Institucionais */}
          <div className="space-y-2.5">
            <h4 className="font-bold text-foreground text-xs uppercase tracking-wider">Conformidade & Transparência</h4>
            <ul className="space-y-1.5">
              <li>
                <a href="#" className="hover:text-primary transition-colors inline-flex items-center gap-1">
                  Lei de Acesso à Informação (LAI)
                </a>
              </li>
              <li>
                <a href="#" className="hover:text-primary transition-colors inline-flex items-center gap-1">
                  Proteção de Dados Pessoais (LGPD)
                </a>
              </li>
              <li>
                <a href="#" className="hover:text-primary transition-colors inline-flex items-center gap-1">
                  Instrução Normativa SCL nº 01/2019
                </a>
              </li>
              <li>
                <Link href="/sobre" className="hover:text-primary transition-colors inline-flex items-center gap-1">
                  Sobre a Plataforma
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-8 pt-6 border-t border-surface-border flex flex-col sm:flex-row items-center justify-between gap-4 text-[11px]">
          <p>© {currentYear} {branding.orgName} — Todos os direitos reservados.</p>
          <p className="font-mono">Desenvolvido com Padrão de Acessibilidade Governamental e-MAG</p>
        </div>
      </div>
    </footer>
  );
}
