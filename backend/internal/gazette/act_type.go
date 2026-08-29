package gazette

import "strings"

// ActType is the single canonical vocabulary for personnel acts extracted from
// the Diário Oficial. Every layer speaks it: the PDF parser that writes
// diario_oficial_findings, the Typesense indexer that fills
// diorondon_personnel_acts, the HTTP query layer and the frontend ACT_TYPES
// list. Before this existed each of those used its own strings
// ("NOMEACAO" vs "NOMEACAO_COMISSIONADO" vs "MUDANCA_SETOR"), so a filter
// picked in the UI matched nothing in storage.
type ActType = string

const (
	ActNomeacaoEfetivo       ActType = "NOMEACAO_EFETIVO"
	ActNomeacaoComissionado  ActType = "NOMEACAO_COMISSIONADO"
	ActContratacaoTemporaria ActType = "CONTRATACAO_TEMPORARIA"
	ActExoneracao            ActType = "EXONERACAO"
	ActRescisao              ActType = "RESCISAO"
	ActRelotacao             ActType = "RELOTACAO"
	ActDesignacaoFuncao      ActType = "DESIGNACAO_FUNCAO"
	ActOutros                ActType = "OUTROS"

	// ActContrato tags contract / "extrato de contrato" findings. They share
	// the diario_oficial_findings table with personnel acts but feed the
	// diorondon_articles collection, not diorondon_personnel_acts.
	ActContrato ActType = "CONTRATO"
)

// CanonicalActTypes is every personnel act_type the UI can filter by, in the
// order the frontend lists them.
var CanonicalActTypes = []ActType{
	ActNomeacaoEfetivo,
	ActNomeacaoComissionado,
	ActContratacaoTemporaria,
	ActExoneracao,
	ActRescisao,
	ActRelotacao,
	ActDesignacaoFuncao,
}

// legacyActTypeAliases maps every value historically emitted by the parsers
// and by hr_parser.HREventType onto the canonical vocabulary, so pre-existing
// PostgreSQL rows and query params sent by older clients still resolve.
var legacyActTypeAliases = map[string]ActType{
	"NOMEACAO":               ActNomeacaoComissionado,
	"NOMEACAO_COMISSIONADA":  ActNomeacaoComissionado,
	"NOMEACAO_COMISSIONADO":  ActNomeacaoComissionado,
	"NOMEACAO_EFETIVO":       ActNomeacaoEfetivo,
	"NOMEACAO_EFETIVA":       ActNomeacaoEfetivo,
	"CONTRATACAO":            ActContratacaoTemporaria,
	"CONTRATACAO_TEMPORARIA": ActContratacaoTemporaria,
	"CONTRATO_TEMPORARIO":    ActContratacaoTemporaria,
	"MUDANCA_SETOR":          ActRelotacao,
	"RELOTACAO":              ActRelotacao,
	"TRANSFERENCIA":          ActRelotacao,
	"REMANEJAMENTO":          ActRelotacao,
	"DESIGNACAO":             ActDesignacaoFuncao,
	"DESIGNACAO_FUNCAO":      ActDesignacaoFuncao,
	"EXONERACAO":             ActExoneracao,
	"RESCISAO":               ActRescisao,
	"CONTRATO":               ActContrato,
	"LICITACAO":              ActContrato,
	"OUTROS":                 ActOutros,
}

// Confidence classifica o grau de certeza da extração de um finding.
// Ver migration 000033. Usado para decidir o que entra na busca do usuário.
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// NormalizeActType maps any legacy or free-form act type onto the canonical
// vocabulary. An empty input stays empty so callers can treat "" as "no
// filter"; an unknown non-empty input falls through to ActOutros.
func NormalizeActType(raw string) ActType {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	if canon, ok := legacyActTypeAliases[s]; ok {
		return canon
	}
	return ActOutros
}
