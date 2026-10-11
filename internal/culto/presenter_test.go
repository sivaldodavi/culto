package culto

import (
	"strings"
	"testing"
)

func TestFormatSlidesUppercasesAndPreservesSlideLines(t *testing.T) {
	got := formatSlides("João 3:16\nDeus amou o mundo\n\noração final")
	if !strings.HasPrefix(got, "JOÃO 3:16\nDEUS AMOU O MUNDO\n\nORAÇÃO FINAL") {
		t.Fatalf("formatação inesperada: %q", got)
	}
	for _, slide := range strings.Split(strings.TrimSpace(got), "\n\n") {
		for _, line := range strings.Split(slide, "\n") {
			if len(line) > slideWidth {
				t.Errorf("linha excede largura: %q", line)
			}
		}
	}
}
