import { Dialog } from "@/components/ui/Dialog";
import { Button } from "@/components/ui/Button";
import { X, FileText, CheckCircle2, Lock, Upload, Printer } from "lucide-react";
import type { DemandResponse } from "@/types/api";

import { useState, useRef } from "react";
import { useToast } from "@/components/notifications/ToastProvider";
import { addDemandDocument } from "@/lib/api/demands";
import Link from "next/link";

interface Props {
  demand: DemandResponse;
  onClose: () => void;
  onDemandUpdated?: () => void;
}

export function ContratoModal({ demand, onClose, onDemandUpdated }: Props) {
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { showToast } = useToast();

  const documents = demand.documents || [];

  const [selectedDocType, setSelectedDocType] = useState<string>("OF_PRE_EMPENHO");

  const docTypeOptions = [
    { value: "OF_PRE_EMPENHO", label: "Ofício de Pré-Empenho / OF" },
    { value: "OFICIO_PLANEJAMENTO", label: "Ofício de Tramitação Planejamento" },
    { value: "EMPENHO_ASSINADO", label: "Nota de Empenho Assinada" },
    { value: "COMPROVANTE_ENVIO", label: "Comprovante de Envio à Empresa" },
    { value: "NOTA_FISCAL", label: "Nota Fiscal / Fatura Atestada" },
    { value: "ORDEM_RECEPCAO", label: "Termo de Recepção / Medição" },
    { value: "RELATORIO_PAGAMENTO", label: "Relatório de Liquidação & Pgto" },
    { value: "CERTIDAO_SIMPLES", label: "Certidão Simplificada / CND" },
    { value: "CERTIDAO_CNDT", label: "Certidão Trabalhista (CNDT)" },
    { value: "CERTIDAO_FGTS", label: "Certificado de Regularidade FGTS" },
  ];

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    try {
      await addDemandDocument(demand.id, selectedDocType, file);
      
      showToast({
        title: "Upload Concluído",
        description: `O documento (${selectedDocType}) ${file.name} foi anexado com sucesso.`,
        tone: "success",
      });
      
      if (onDemandUpdated) onDemandUpdated();
      
    } catch (err: any) {
      showToast({
        title: "Erro no Upload",
        description: err.message || "Não foi possível enviar o documento para o storage.",
        tone: "danger",
      });
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };
  
  return (
    <Dialog 
      open={true} 
      onClose={onClose}
      title={`Demanda: ${demand.ano_mes}`}
      description={`Contrato ${demand.contrato_numero} • ${demand.contratado}`}
    >
        <div className="grid gap-6 py-2">
          <div className="flex flex-col gap-2">
            <h4 className="text-sm font-medium text-foreground">Objeto do Contrato</h4>
            <p className="text-sm text-muted rounded-md bg-surface-hover/30 p-3 border border-border/50">
              {demand.contrato_objeto}
            </p>
          </div>

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-2">
              <h4 className="text-sm font-medium text-foreground flex items-center gap-2">
                <FileText className="h-4 w-4 text-primary" />
                Documentos Exigidos
              </h4>
              
              <div className="flex items-center gap-2">
                <select
                  value={selectedDocType}
                  onChange={(e) => setSelectedDocType(e.target.value)}
                  className="rounded-md border border-surface-border bg-surface px-2 py-1 text-xs font-medium text-foreground focus:border-primary focus:outline-none"
                >
                  {docTypeOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>

                <input 
                  type="file" 
                  ref={fileInputRef} 
                  onChange={handleFileChange} 
                  className="hidden" 
                  accept="application/pdf,image/*" 
                />
                <Button size="sm" variant="secondary" className="h-8 text-xs shrink-0" onClick={handleUploadClick} disabled={isUploading}>
                  <Upload className={`mr-2 h-3 w-3 ${isUploading ? "animate-pulse" : ""}`} />
                  {isUploading ? "Enviando..." : "Anexar"}
                </Button>
              </div>
            </div>

            <div className="rounded-md border border-border/50 divide-y divide-border/50">
              {documents.length > 0 ? (
                documents.map((doc) => (
                  <div key={doc.id} className="flex items-center justify-between p-3 bg-surface-hover/10">
                    <div className="flex items-center gap-3">
                      <CheckCircle2 className="h-4 w-4 text-success" />
                      <div className="flex flex-col">
                        <span className="text-sm font-medium">{doc.doc_type}</span>
                        <span className="text-xs text-muted">{doc.file_name}</span>
                      </div>
                    </div>
                  </div>
                ))
              ) : (
                <div className="flex flex-col items-center justify-center p-6 text-center text-muted">
                  <Lock className="h-8 w-8 mb-2 text-danger/70" />
                  <p className="text-sm">Nenhum documento anexado.</p>
                  <p className="text-xs">O Kanban está bloqueado até o envio dos comprovantes.</p>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="flex justify-between border-t border-border/40 pt-4 mt-4">
          <Link href={`/contratos/demandas/${demand.id}/oficio`} target="_blank">
            <Button variant="secondary" className="gap-2">
              <Printer className="h-4 w-4" />
              Gerar OF (PDF)
            </Button>
          </Link>
          <Button variant="secondary" onClick={onClose}>
            Fechar
          </Button>
        </div>
    </Dialog>
  );
}
