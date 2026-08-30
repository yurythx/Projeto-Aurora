package gazette

import "strings"

// EditionType classifica uma edição do Diário Oficial. Assim como ActType,
// é vocabulário canônico único — o parser de PDF, o reindexador do
// Typesense e o schema das coleções (edition_type, facet) falam o mesmo.
// Antes o parser cravava "ORDINARIA" e só o reindexador classificava de
// verdade, então uma edição suplementar/extra parseada direto ficava
// rotulada errada.
type EditionType = string

const (
	EditionOrdinaria   EditionType = "ORDINARIA"
	EditionSuplementar EditionType = "SUPLEMENTAR"
)

// EditionTypeFromNumber deriva o tipo do número da edição: as suplementares
// / extras do DIORONDON vêm com sufixo "S" ou "E" (ex.: "6262S"), ou o
// texto "SUPLEMENT"/"EXTRA" embutido. Sem nenhum desses marcadores, é
// ordinária.
func EditionTypeFromNumber(editionNumber string) EditionType {
	up := strings.ToUpper(strings.TrimSpace(editionNumber))
	if strings.HasSuffix(up, "S") || strings.HasSuffix(up, "E") ||
		strings.Contains(up, "SUPLEMENT") || strings.Contains(up, "EXTRA") {
		return EditionSuplementar
	}
	return EditionOrdinaria
}
