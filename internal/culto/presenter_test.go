package culto

import (
	"os"
	"path/filepath"
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

func TestNumberedSlidesPreviewUsesTheProjectedSlideBoundaries(t *testing.T) {
	got := NumberedSlidesPreview("capa\n\nprimeira linha\nsegunda linha\n\núltimo")
	want := "SLIDE 01\nCAPA\n\nSLIDE 02\nPRIMEIRA LINHA\nSEGUNDA LINHA\n\nSLIDE 03\nÚLTIMO"
	if got != want {
		t.Fatalf("pré-visualização numerada:\n%s\nesperada:\n%s", got, want)
	}
}

func TestJumpToSlideSendsSentGSequenceToWindow(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	xdotool := filepath.Join(dir, "xdotool")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = search ]; then printf '4242\\n'; exit 0; fi\n" +
		"printf '%s\\n' \"$@\" > \"$CULTO_TEST_XDOTOOL_ARGS\"\n"
	if err := os.WriteFile(xdotool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CULTO_TEST_XDOTOOL_ARGS", argsFile)

	if err := JumpToSlide(12); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "key\n--window\n4242\n--clearmodifiers\ng\n1\n2\nReturn\n"
	if string(got) != want {
		t.Fatalf("argumentos enviados ao xdotool:\n%s\nesperado:\n%s", got, want)
	}
}

func TestJumpToSlideRejectsZero(t *testing.T) {
	if err := JumpToSlide(0); err == nil {
		t.Fatal("slide zero deveria ser rejeitado")
	}
}
