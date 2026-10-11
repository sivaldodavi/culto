package culto

import (
	"strings"
	"testing"
)

func testBible() []BibleBook {
	return []BibleBook{
		{Name: "Gênesis", Abbrev: "Gn", Chapters: [][]string{
			{"No princípio.", "A terra era sem forma."},
			{"Deus fez a luz."},
		}},
		{Name: "João", Abbrev: "Jo", Chapters: [][]string{
			{"No princípio era o Verbo.", "O Verbo estava com Deus."},
		}},
		{Name: "Jonas", Abbrev: "Jn", Chapters: [][]string{{"Jonas fugiu."}}},
	}
}

func TestParseReferenceByBookChapterAndRange(t *testing.T) {
	verses, label, err := ParseReference("Gn 1:1-2", testBible())
	if err != nil {
		t.Fatal(err)
	}
	if label != "Gênesis 1:1-2" {
		t.Fatalf("rótulo %q", label)
	}
	if len(verses) != 2 || verses[1].Text != "A terra era sem forma." {
		t.Fatalf("resultado inesperado: %#v", verses)
	}
}

func TestParseReferenceRejectsInvalidInput(t *testing.T) {
	for _, query := range []string{" ", "J 1", "Gênesis 9", "Gn 1:3", "João 1:2-1"} {
		t.Run(query, func(t *testing.T) {
			if _, _, err := ParseReference(query, testBible()); err == nil {
				t.Fatalf("esperava erro para %q", query)
			}
		})
	}
}

func TestParseReferenceExactAbbreviationWinsOverPrefix(t *testing.T) {
	verses, _, err := ParseReference("Jo 1:1", testBible())
	if err != nil {
		t.Fatal(err)
	}
	if len(verses) != 1 || verses[0].Book != "João" {
		t.Fatalf("resultado inesperado: %#v", verses)
	}
}

func TestSearchBibleRequiresQueryAndFindsMatches(t *testing.T) {
	if _, err := SearchBible(testBible(), " "); err == nil {
		t.Fatal("busca vazia deveria falhar")
	}
	results, err := SearchBible(testBible(), "VERBO")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("encontradas %d ocorrências, esperado 2", len(results))
	}
}

func TestRenderBibleResponsiveAndStandard(t *testing.T) {
	verses, label, err := ParseReference("João 1:1-2", testBible())
	if err != nil {
		t.Fatal(err)
	}
	standard := RenderBible(verses, label, false, false)
	if !strings.Contains(standard, "JOÃO 1:1-2") || !strings.Contains(standard, "1. No princípio") {
		t.Fatalf("leitura comum inesperada: %q", standard)
	}
	responsive := RenderBible(verses, label, false, true)
	if !strings.Contains(responsive, "(DIRIGENTE)") || !strings.Contains(responsive, "(IGREJA)") {
		t.Fatalf("leitura responsiva inesperada: %q", responsive)
	}
}
