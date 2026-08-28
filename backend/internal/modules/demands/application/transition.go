package application

import (
	"fmt"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// ValidateTransition encapsula a Máquina de Estados do Kanban.
// Ele verifica se a transição da etapa atual (current) para a desejada (next) 
// é permitida com base na posse dos documentos anexados.
func ValidateTransition(demand domain.MonthlyDemand, nextStage domain.EtapaKanban) error {
	// Apenas permite avançar para a próxima etapa exata (+1) ou voltar etapas.
	if nextStage > demand.Etapa+1 {
		return fmt.Errorf("não é permitido pular etapas (tentou avançar de %d para %d)", demand.Etapa, nextStage)
	}
	
	// Se está regredindo de etapa, é sempre permitido (ex: voltar para correção).
	if nextStage <= demand.Etapa {
		return nil
	}

	// map de documentos anexados para busca rápida O(1)
	docs := make(map[domain.DocumentType]bool)
	for _, d := range demand.Documents {
		docs[d.DocType] = true
	}

	switch demand.Etapa {
	case domain.Etapa1ElaborarOF:
		// Para ir da Etapa 1 -> 2:
		// "Regra de Bloqueio: O card não avança se faltar qualquer um dos 3 documentos anexados."
		if !docs[domain.DocOFPreEmpenho] || !docs[domain.DocOficioPlanej] {
			return fmt.Errorf("transição bloqueada: faltam documentos obrigatórios da Etapa 1 (OF / Pré-Empenho / Ofício)")
		}

	case domain.Etapa2TramitarPlan:
		// Para ir da Etapa 2 -> 3: 
		// Nenhum documento novo exigido, apenas o tempo de espera.
		break

	case domain.Etapa3EmitirOS:
		// Para ir da Etapa 3 -> 4:
		// "Regra de Bloqueio: Só avança após upload do Empenho assinado e confirmação formal do envio"
		if !docs[domain.DocEmpenhoAssinado] || !docs[domain.DocComprovanteEnvio] {
			return fmt.Errorf("transição bloqueada: falta o Empenho Assinado ou o Comprovante de Envio ao fornecedor")
		}

	case domain.Etapa4ExecucaoRecepcao:
		// Para ir da Etapa 4 -> 5:
		// "Regra de Bloqueio: Travado até upload de ambos os arquivos (Nota Fiscal e Ordem de Recepção)."
		if !docs[domain.DocNotaFiscal] || !docs[domain.DocOrdemRecepcao] {
			return fmt.Errorf("transição bloqueada: aguardando entrega do fornecedor (Nota Fiscal e Ordem de Recepção pendentes)")
		}

	case domain.Etapa5RelatorioPgto:
		// Para ir da Etapa 5 -> 6:
		// "Regra de Bloqueio: Faltando certidões válidas ou DAM/Medição, o avanço é bloqueado."
		requiredCertidoes := []domain.DocumentType{
			domain.DocExtratoEmpenho, domain.DocRelatorioPgto,
			domain.DocCertidaoSimples, domain.DocCertidaoCNDT, domain.DocCertidaoFGTS,
			domain.DocCertidaoMunicipal, domain.DocCertidaoEstadual, domain.DocCertidaoFederal,
		}

		for _, req := range requiredCertidoes {
			if !docs[req] {
				return fmt.Errorf("transição bloqueada: falta o documento obrigatório '%s' para compliance fiscal", req)
			}
		}

		// Regra especial para Serviços e Obras
		if demand.ContractType == domain.ServicosTerceirizados || demand.ContractType == domain.Obras {
			if !docs[domain.DocGuiaDAMISSQN] || !docs[domain.DocPlanilhaMedicao] {
				return fmt.Errorf("transição bloqueada: contratos de serviço exigem Guia DAM (ISSQN) e Planilha de Medição anexadas")
			}
		}
	}

	return nil
}
