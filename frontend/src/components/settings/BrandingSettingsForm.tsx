"use client";

import { useState } from "react";
import { useBranding, DEFAULT_BRANDING, type SystemBrandingConfig } from "@/components/branding/BrandingContext";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { useToast } from "@/components/notifications/ToastProvider";
import { Building2, Save, RotateCcw, ShieldCheck } from "lucide-react";

export function BrandingSettingsForm() {
  const { branding, updateBranding, resetBranding } = useBranding();
  const { showToast } = useToast();

  const [form, setForm] = useState<SystemBrandingConfig>(branding);
  const [logoPreviewError, setLogoPreviewError] = useState(false);

  // Ressincroniza o formulário quando o branding do contexto muda (ex.:
  // após salvar / restaurar padrões). Ajuste de estado durante o render —
  // padrão recomendado pelo React em vez de um useEffect com setState.
  const [syncedBranding, setSyncedBranding] = useState(branding);
  if (branding !== syncedBranding) {
    setSyncedBranding(branding);
    setForm(branding);
  }

  const handleChange = (field: keyof SystemBrandingConfig, value: string | boolean) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    if (field === "logoUrl") setLogoPreviewError(false);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    updateBranding(form);
    showToast({
      title: "Configurações Salvas",
      description: "A identidade visual e os metadados da aplicação foram atualizados.",
      tone: "success",
    });
  };

  const handleReset = () => {
    if (confirm("Deseja restaurar as configurações padrão da aplicação?")) {
      resetBranding();
      setForm(DEFAULT_BRANDING);
      showToast({
        title: "Padrões Restaurados",
        description: "A identidade visual padrão foi reestabelecida.",
        tone: "info",
      });
    }
  };

  return (
    <Card className="border border-surface-border bg-surface">
      <CardHeader className="border-b border-surface-border">
        <CardTitle className="text-base font-bold text-foreground flex items-center gap-2">
          <Building2 className="h-5 w-5 text-primary" />
          Identidade Visual Institucional & Branding White-Label
        </CardTitle>
        <p className="text-xs text-muted mt-1">
          Personalize o nome da aplicação, logomarca, descrição e dados de contato do órgão municipal.
        </p>
      </CardHeader>

      <CardContent className="p-6">
        <form onSubmit={handleSave} className="space-y-6 text-xs">
          {/* Seção 1: Identificação da Aplicação */}
          <div className="space-y-4">
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary border-b border-surface-border pb-1">
              1. Identificação & Metadados
            </h3>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block font-medium text-foreground mb-1">Nome da Aplicação / Sistema *</label>
                <Input
                  value={form.appName}
                  onChange={(e) => handleChange("appName", e.target.value)}
                  placeholder="Ex: Projeto Nova"
                  required
                />
              </div>

              <div>
                <label className="block font-medium text-foreground mb-1">Órgão / Prefeitura Municipal *</label>
                <Input
                  value={form.orgName}
                  onChange={(e) => handleChange("orgName", e.target.value)}
                  placeholder="Ex: Prefeitura Municipal de Rondonópolis"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block font-medium text-foreground mb-1">Descrição Institucional</label>
              <textarea
                value={form.appDescription}
                onChange={(e) => handleChange("appDescription", e.target.value)}
                rows={2}
                className="w-full rounded border border-surface-border bg-surface p-2 text-foreground text-xs focus:border-primary focus:outline-none"
                placeholder="Descrição curta para a barra e-MAG e metadados..."
              />
            </div>
          </div>

          {/* Seção 2: Logomarca & Favicon (Imagens Seguras) */}
          <div className="space-y-4">
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary border-b border-surface-border pb-1">
              2. Logomarca & Imagens Institucionais
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-3">
                <label className="block font-medium text-foreground">URL da Logomarca (PNG/SVG/WebP)</label>
                <Input
                  value={form.logoUrl}
                  onChange={(e) => handleChange("logoUrl", e.target.value)}
                  placeholder="https://exemplo.gov.br/logo.png"
                />
                <p className="text-[11px] text-muted">Cole o link direto da imagem pública da prefeitura.</p>
              </div>

              {/* Preview em Tempo Real */}
              <div className="rounded-lg border border-surface-border bg-surface-hover/30 p-4 flex flex-col items-center justify-center min-h-[100px]">
                <span className="text-[10px] uppercase font-bold text-muted mb-2">Pré-visualização da Logomarca</span>
                {form.logoUrl && !logoPreviewError ? (
                  // URL externa digitada pelo órgão; next/image exige domínio pré-configurado.
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={form.logoUrl}
                    alt="Preview da Logo"
                    onError={() => setLogoPreviewError(true)}
                    className="h-10 w-auto object-contain max-w-[180px]"
                  />
                ) : (
                  <div className="flex items-center gap-2 text-muted">
                    <ShieldCheck className="h-6 w-6 text-primary" />
                    <span className="text-xs font-semibold">{form.appName || "Projeto Nova"}</span>
                    <span className="text-[10px] text-muted">(Fallback Vetorial)</span>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Seção 3: Suporte & Contatos */}
          <div className="space-y-4">
            <h3 className="text-xs font-bold uppercase tracking-wider text-primary border-b border-surface-border pb-1">
              3. Canais de Atendimento & Suporte
            </h3>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block font-medium text-foreground mb-1">E-mail de Suporte</label>
                <Input
                  type="email"
                  value={form.supportEmail}
                  onChange={(e) => handleChange("supportEmail", e.target.value)}
                  placeholder="suporte@municipio.gov.br"
                />
              </div>

              <div>
                <label className="block font-medium text-foreground mb-1">Telefone de Atendimento</label>
                <Input
                  value={form.supportPhone}
                  onChange={(e) => handleChange("supportPhone", e.target.value)}
                  placeholder="(66) 3411-5000"
                />
              </div>

              <div>
                <label className="block font-medium text-foreground mb-1">Horário de Atendimento</label>
                <Input
                  value={form.supportHours}
                  onChange={(e) => handleChange("supportHours", e.target.value)}
                  placeholder="Segunda a Sexta, 08h às 17h"
                />
              </div>
            </div>
          </div>

          {/* Botões de Ação */}
          <div className="flex items-center justify-between pt-4 border-t border-surface-border">
            <Button type="button" variant="ghost" onClick={handleReset} className="text-danger hover:bg-danger/10">
              <RotateCcw className="mr-1.5 h-4 w-4" />
              Restaurar Padrões
            </Button>

            <Button type="submit" variant="primary">
              <Save className="mr-1.5 h-4 w-4" />
              Salvar Alterações
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
