package gazette

import "testing"

func TestEditionTypeFromNumber(t *testing.T) {
	cases := []struct {
		in   string
		want EditionType
	}{
		{"6262", EditionOrdinaria},
		{"6262S", EditionSuplementar},
		{"6262s", EditionSuplementar},
		{" 6262E ", EditionSuplementar},
		{"Edição 6262 - SUPLEMENTAR", EditionSuplementar},
		{"6262 EXTRA", EditionSuplementar},
		{"", EditionOrdinaria},
		{"7000", EditionOrdinaria},
	}
	for _, c := range cases {
		if got := EditionTypeFromNumber(c.in); got != c.want {
			t.Errorf("EditionTypeFromNumber(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
