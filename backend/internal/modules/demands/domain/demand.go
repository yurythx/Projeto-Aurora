package domain

import (
	"time"

	"github.com/google/uuid"
)

// EtapaKanban representa um dos 6 passos definidos na IN SCL 01/2019 e IN SCL 04/2021.
type EtapaKanban int

const (
	Etapa1ElaborarOF       EtapaKanban = 1
	Etapa2TramitarPlan     EtapaKanban = 2
	Etapa3EmitirOS         EtapaKanban = 3
	Etapa4ExecucaoRecepcao EtapaKanban = 4
	Etapa5RelatorioPgto    EtapaKanban = 5
	Etapa6Contabilidade    EtapaKanban = 6
)

// Label devolve o rótulo de exibição da etapa (colunas do Kanban / funil
// do dashboard).
func (e EtapaKanban) Label() string {
	switch e {
	case Etapa1ElaborarOF:
		return "Elaborar OF / Pré-Empenho"
	case Etapa2TramitarPlan:
		return "Tramitar Planejamento"
	case Etapa3EmitirOS:
		return "Emitir OS / Envio Empresa"
	case Etapa4ExecucaoRecepcao:
		return "Execução e Recepção"
	case Etapa5RelatorioPgto:
		return "Relatório Pgto / Certidões"
	case Etapa6Contabilidade:
		return "Contabilidade"
	default:
		return "Etapa desconhecida"
	}
}

// StatusEtapa indica o status interno do card na coluna atual do Kanban.
type StatusEtapa string

const (
	StatusPendente            StatusEtapa = "pendente"
	StatusAguardandoDocumento StatusEtapa = "aguardando_documento"
	StatusConcluida           StatusEtapa = "concluida"
)

// ContractType afeta quais documentos são exigidos.
type ContractType string

const (
	CompraConsumo         ContractType = "COMPRA_CONSUMO"
	ServicosTerceirizados ContractType = "SERVICOS_TERCEIRIZADOS"
	Obras                 ContractType = "OBRAS"
)

// DocumentType especifica todos os tipos de documentos mapeados no roadmap.
type DocumentType string

const (
	DocOFPreEmpenho      DocumentType = "OF_PRE_EMPENHO"
	DocOficioPlanej      DocumentType = "OFICIO_PLANEJAMENTO"
	DocEmpenhoAssinado   DocumentType = "EMPENHO_ASSINADO"
	DocComprovanteEnvio  DocumentType = "COMPROVANTE_ENVIO"
	DocNotaFiscal        DocumentType = "NOTA_FISCAL"
	DocOrdemRecepcao     DocumentType = "ORDEM_RECEPCAO"
	DocExtratoEmpenho    DocumentType = "EXTRATO_EMPENHO"
	DocRelatorioPgto     DocumentType = "RELATORIO_PAGAMENTO"
	DocCertidaoSimples   DocumentType = "CERTIDAO_SIMPLES"
	DocCertidaoCNDT      DocumentType = "CERTIDAO_CNDT"
	DocCertidaoFGTS      DocumentType = "CERTIDAO_FGTS"
	DocCertidaoMunicipal DocumentType = "CERTIDAO_MUNICIPAL"
	DocCertidaoEstadual  DocumentType = "CERTIDAO_ESTADUAL"
	DocCertidaoFederal   DocumentType = "CERTIDAO_FEDERAL"
	DocGuiaDAMISSQN      DocumentType = "GUIA_DAM_ISSQN"
	DocPlanilhaMedicao   DocumentType = "PLANILHA_MEDICAO"
)

// MonthlyDemand (Demanda Mensal) é o Card do Kanban de liquidação de pagamento.
type MonthlyDemand struct {
	ID             uuid.UUID
	ContratoID     uuid.UUID
	AnoMes         string // ex: "2026-08"
	Etapa          EtapaKanban
	StatusEtapa    StatusEtapa
	Observacoes    string
	EtapaStartedAt time.Time
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// Relacionamentos carregados em cache/query:
	Documents      []DemandDocument
	ContractType   ContractType
	ContratoNumero string
	ContratoObjeto string
	Contratado     string
	ContratoValor  *float64
}

// DemandDocument representa um arquivo anexado à Demanda (uploads/provas).
type DemandDocument struct {
	ID         uuid.UUID
	DemandaID  uuid.UUID
	DocType    DocumentType
	FilePath   string
	FileName   string
	UploadedBy *uuid.UUID
	UploadedAt time.Time
	Validade   *time.Time
}

// Ocurrence (Antingerência IN 04/2021) mapeia as notificações ao preposto.
type Occurrence struct {
	ID         uuid.UUID
	ContratoID uuid.UUID
	DemandaID  *uuid.UUID
	Tipo       string
	Descricao  string
	SLAVenceEm *time.Time
	Resolvida  bool
	CreatedBy  *uuid.UUID
	CreatedAt  time.Time
}
