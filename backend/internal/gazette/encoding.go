package gazette

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// RepairEncoding devolve uma versão UTF-8 VÁLIDA de s.
//
// Contexto: o pdftotext, quando o PDF traz uma fonte com encoding embutido
// não-padrão e sem ToUnicode CMap, emite bytes crus da tabela da fonte
// (Latin-1 / Windows-1252) — bytes altos isolados como 0xA2 ("¢"), 0xB3 ("³"),
// 0xBA ("º"). Isolados, são UTF-8 inválido, e o PostgreSQL (banco UTF-8)
// rejeita o INSERT com SQLSTATE 22021, abortando a transação inteira da
// edição. Foi o que derrubou 2 edições para FAILED com zero findings.
//
// Estratégia: se s já é UTF-8 válido, retorna intacto (caminho rápido). Se
// não, cada byte inválido é reinterpretado como Windows-1252 (superconjunto do
// Latin-1, o encoding mais comum nesses PDFs) e transcodificado — assim
// "¢ ³ º ª § ½ –" sobrevivem, em vez de serem deletados (deletar pode colar
// dois tokens: "R$ 1.234" + byte solto vira "R$ 1.234...").
func RepairEncoding(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 16)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(charmap.Windows1252.DecodeByte(s[i]))
			i++
			continue
		}
		b.WriteString(s[i : i+size])
		i += size
	}

	out := b.String()
	// Windows-1252 deixa 5 posições sem mapa (0x81, 0x8D, 0x8F, 0x90, 0x9D);
	// se alguma sobrou, remove no final para garantir UTF-8 válido.
	if !utf8.ValidString(out) {
		out = strings.ToValidUTF8(out, "")
	}
	return out
}
