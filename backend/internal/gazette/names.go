package gazette

import (
	"regexp"
	"strings"
)

var nameConnectives = map[string]bool{
	"de": true, "da": true, "do": true, "dos": true, "das": true, "e": true,
}

// IsPlausiblePersonName aplica os invariantes mínimos de um nome de pessoa
// física: 2 a 7 palavras, pelo menos 2 delas NÃO conectivos (recusa "de Grosso",
// "das Quantidades", "DE AGOSTO DE"), nenhum dígito, e não parece razão
// social / órgão / ato de licitação (LooksLikeOrganization).
func IsPlausiblePersonName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "0123456789") {
		return false
	}
	words := strings.Fields(name)
	if len(words) < 2 || len(words) > 7 {
		return false
	}
	real := 0
	for _, w := range words {
		if !nameConnectives[strings.ToLower(w)] {
			real++
		}
	}
	if real < 2 {
		return false
	}
	return !LooksLikeOrganization(name)
}

// orgTokenRegex casa marcadores que denunciam razão social de empresa, órgão
// público ou cabeçalho de página — nunca um nome de pessoa física. É usado
// para descartar capturas erradas das regexes gulosas de nomeação/exoneração
// (que pegavam "AGROBEN COMÉRCIO DE PRODUTOS AGROPECUÁRIOS LTDA" ou
// "DIÁRIO OFICIAL ELETRÔNICO DE RONDONÓPOLIS" como se fossem servidores).
var orgTokenRegex = regexp.MustCompile(`(?i)\b(` +
	`LTDA|EIRELI|EPP|MEI|S/?A|CIA|COMPANHIA|` +
	`COM[ÉE]RCIO|IND[ÚU]STRIA|DISTRIBUIDORA|CONSTRU[ÇC][ÕO]ES|TRANSPORTES|` +
	`SERVI[ÇC]OS|SOLU[ÇC][ÕO]ES|TECNOLOGIA|ENGENHARIA|CONSULTORIA|` +
	`ASSOCIA[ÇC][ÃA]O|FUNDA[ÇC][ÃA]O|COOPERATIVA|SINDICATO|INSTITUTO|` +
	`SECRETARIA|PREFEITURA|MUNIC[ÍI]PIO|C[ÂA]MARA|CONSELHO|TRIBUNAL|MINIST[ÉE]RIO|` +
	`FUNDO|AUTARQUIA|GAB\.?|GABINETE|DI[ÁA]RIO\s+OFICIAL|AGROPECU[ÁA]RI|` +
	`AG[ÊE]NCIA|IMOBILI[ÁA]RIA|CONSTRUTORA|NUTRI[ÇC][ÃA]O|PRODUTOS|MEDKA|` +
	`ESTADO\s+DE|UNI[ÃA]O|REP[ÚU]BLICA|PODER\s+|REPRESENTANTES|CASA\s+DE\s+APOIO|` +
	// marcadores de ato de licitação/compras / peças processuais
	`PROCESSO|LICITA[ÇC][ÃA]O|DISPENSA|INEXIGIBILIDADE|PREG[ÃA]O|RATIFICA|HOMOLOGA|CHAMAMENTO|CREDENCIAMENTO|` +
	`TERMO\s+ADITIVO|ADITIVO|RECURSO|DEFERIMENTO|INDEFERIMENTO|INSCRITOS|LEI\s+(MUNICIPAL|COMPLEMENTAR|FEDERAL)|` +
	`FISCAL\s+(TITULAR|SUPLENTE)|GT\s+|CIB` +
	`)\b`)

var cnpjInStringRegex = regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}\b`)

// nameNoiseRegex casa substantivos abstratos de procedimento que o parser de
// blocos às vezes captura como "nome" ("DIVULGAÇÃO DOS LOCAIS DE PROVA",
// "MANIFESTAÇÃO DE INTERESSE"). Nenhuma pessoa se chama assim.
var nameNoiseRegex = regexp.MustCompile(`(?i)\b(` +
	`DIVULGA[ÇC][ÃA]O|MANIFESTA[ÇC][ÃA]O|INTERESSE|LOCAIS|PROVA|RESULTADO|` +
	`INSCRI[ÇC][ÃA]O|CLASSIFICA[ÇC][ÃA]O|CONVOCA[ÇC][ÃA]O|RETIFICA[ÇC][ÃA]O|` +
	`HOMOLOGA[ÇC][ÃA]O|CONCURSO|SELETIVO|ANEXO|CRONOGRAMA|QUANTIDADE|VALIDADE|` +
	`DISPOSI[ÇC][ÕO]ES|FINAIS|VIG[ÊE]NCIA|PRAZO|OBJETO|` +
	`AGRICULTURA|PECU[ÁA]RIA|ATEN[ÇC][ÃA]O|INTEGRA[ÇC][ÃA]O|ACOMPANHAMENTO|` +
	`COMISS[ÃA]O|COMIT[ÊE]|GRUPO\s+DE\s+TRABALHO|EQUIPE|` +
	// títulos de autoridade signatária (não são o nome do servidor do ato)
	`PREFEITO|SECRET[ÁA]RI[OA]|GOVERNADOR|PROCURADOR-GERAL` +
	`)\b`)

// LooksLikeOrganization retorna true se s parece ser razão social, órgão,
// cabeçalho ou fragmento de procedimento — não uma pessoa física.
func LooksLikeOrganization(s string) bool {
	return orgTokenRegex.MatchString(s) || cnpjInStringRegex.MatchString(s) || nameNoiseRegex.MatchString(s)
}
