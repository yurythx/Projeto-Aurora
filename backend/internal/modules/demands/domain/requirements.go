package domain

// Label é o rótulo de exibição de cada tipo de documento.
func (d DocumentType) Label() string {
	switch d {
	case DocOFPreEmpenho:
		return "Ordem de Fornecimento / Pré-Empenho"
	case DocOficioPlanej:
		return "Ofício ao Planejamento"
	case DocEmpenhoAssinado:
		return "Nota de Empenho Assinada"
	case DocComprovanteEnvio:
		return "Comprovante de Envio ao Fornecedor"
	case DocNotaFiscal:
		return "Nota Fiscal / Fatura Atestada"
	case DocOrdemRecepcao:
		return "Ordem de Recepção (Sistema Legado)"
	case DocExtratoEmpenho:
		return "Extrato de Empenho"
	case DocRelatorioPgto:
		return "Relatório de Pagamento"
	case DocCertidaoSimples:
		return "Certidão Simplificada / CND"
	case DocCertidaoCNDT:
		return "Certidão Negativa de Débitos Trabalhistas (CNDT)"
	case DocCertidaoFGTS:
		return "Certidão de Regularidade do FGTS"
	case DocCertidaoMunicipal:
		return "Certidão Negativa Municipal"
	case DocCertidaoEstadual:
		return "Certidão Negativa Estadual"
	case DocCertidaoFederal:
		return "Certidão Negativa Federal"
	case DocGuiaDAMISSQN:
		return "Guia DAM (ISSQN)"
	case DocPlanilhaMedicao:
		return "Planilha de Medição"
	default:
		return string(d)
	}
}

// IsCertidao reporta se o documento é uma das certidões de compliance que têm
// prazo de validade (bloqueiam a demanda quando vencidas, IN SCL 01/2019).
func (d DocumentType) IsCertidao() bool {
	switch d {
	case DocCertidaoSimples, DocCertidaoCNDT, DocCertidaoFGTS,
		DocCertidaoMunicipal, DocCertidaoEstadual, DocCertidaoFederal:
		return true
	default:
		return false
	}
}

// RequiredDocsForAdvance devolve os documentos exigidos para SAIR de fromEtapa
// (ou seja, para avançar para fromEtapa+1). contractType adiciona os documentos
// cumulativos de contratos de serviço/obra na Etapa 5.
func RequiredDocsForAdvance(fromEtapa EtapaKanban, contractType ContractType) []DocumentType {
	switch fromEtapa {
	case Etapa1ElaborarOF:
		return []DocumentType{DocOFPreEmpenho, DocOficioPlanej}
	case Etapa2TramitarPlan:
		return nil // espera externa — sem documento novo
	case Etapa3EmitirOS:
		return []DocumentType{DocEmpenhoAssinado, DocComprovanteEnvio}
	case Etapa4ExecucaoRecepcao:
		return []DocumentType{DocNotaFiscal, DocOrdemRecepcao}
	case Etapa5RelatorioPgto:
		req := []DocumentType{
			DocExtratoEmpenho, DocRelatorioPgto,
			DocCertidaoSimples, DocCertidaoCNDT, DocCertidaoFGTS,
			DocCertidaoMunicipal, DocCertidaoEstadual, DocCertidaoFederal,
		}
		if contractType == ServicosTerceirizados || contractType == Obras {
			req = append(req, DocGuiaDAMISSQN, DocPlanilhaMedicao)
		}
		return req
	default:
		return nil
	}
}
