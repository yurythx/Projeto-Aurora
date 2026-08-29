package gazette

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRepairEncoding(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"utf8 válido passa intacto", "Servidor João Conceição — R$ 9.800,00", "Servidor João Conceição — R$ 9.800,00"},
		{"vazio", "", ""},
		// bytes que derrubaram as edições 6256/6264: 0xA2 (¢) e 0xB3 (³) isolados
		{"0xA2 latin1 -> ¢", "valor de 50\xa2 por hora", "valor de 50¢ por hora"},
		{"0xB3 latin1 -> ³", "area de 30m\xb3", "area de 30m³"},
		{"0xBA latin1 -> º", "Portaria n\xba 41.735", "Portaria nº 41.735"},
		{"múltiplos bytes ruins", "\xaa\xba\xbd fim", "ªº½ fim"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RepairEncoding(tc.in)
			if !utf8.ValidString(got) {
				t.Fatalf("saída não é UTF-8 válido: %q", got)
			}
			if got != tc.want {
				t.Errorf("RepairEncoding(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRepairEncoding_NeverPanicsOnAnyByte varre todos os 256 valores de byte
// isolados — nenhum pode causar panic nem produzir UTF-8 inválido.
func TestRepairEncoding_NeverPanicsOnAnyByte(t *testing.T) {
	for b := 0; b < 256; b++ {
		s := "x" + string([]byte{byte(b)}) + "y"
		out := RepairEncoding(s)
		if !utf8.ValidString(out) {
			t.Errorf("byte 0x%02X -> UTF-8 inválido: %q", b, out)
		}
		if !strings.HasPrefix(out, "x") || !strings.HasSuffix(out, "y") {
			t.Errorf("byte 0x%02X: âncoras x..y perdidas: %q", b, out)
		}
	}
}
