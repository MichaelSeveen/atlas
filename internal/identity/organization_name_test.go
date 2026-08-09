package identity

import "testing"

func TestOrganizationNameNormalizationUsesNFKCCaseFold(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, input, want string
	}{
		{"ascii", "  Atlas   Treasury  ", "atlas treasury"},
		{"fullwidth", "ＡＴＬＡＳ", "atlas"},
		{"case-fold", "Straße", "strasse"},
		{"canonical-composition", "A\u030Aland", "åland"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeOrganizationName(test.input); got != test.want {
				t.Fatalf("normalizeOrganizationName(%q)=%q want=%q", test.input, got, test.want)
			}
		})
	}
}
