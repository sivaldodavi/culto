package culto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveSongPreservesStanzasAndRendersHymnNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "musicas.json")
	title := "12-A - Hino de teste"
	if err := SaveSong(path, title, "Primeira estrofe\ncontinua\n\nSegunda estrofe"); err != nil {
		t.Fatal(err)
	}
	songs, err := LoadSongs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(songs["Hinos"][title]); got != 2 {
		t.Fatalf("foram salvas %d estrofes, esperado 2", got)
	}
	if got := RenderSong(title, songs["Hinos"][title]); !strings.HasPrefix(got, "HINO 12-A") {
		t.Fatalf("cabeçalho inesperado: %q", got)
	}
}

func TestNumberedSongPreviewMatchesPresentationSlideNumbers(t *testing.T) {
	title := "12-A - Hino de teste"
	stanzas := []string{"Primeira estrofe", "Segunda estrofe"}

	preview := NumberedSongPreview(title, stanzas)
	for _, want := range []string{
		"SLIDE 01 · CAPA\nHINO 12-A",
		"SLIDE 02 · ESTROFE 1\nPrimeira estrofe",
		"SLIDE 03 · ESTROFE 2\nSegunda estrofe",
	} {
		if !strings.Contains(preview, want) {
			t.Errorf("pré-visualização não contém %q:\n%s", want, preview)
		}
	}
	if projected := RenderSong(title, stanzas); strings.Contains(projected, "SLIDE 0") ||
		strings.Contains(projected, "ESTROFE 1") {
		t.Fatalf("rótulos da pré-visualização vazaram para a projeção: %q", projected)
	}
}

func TestNumberedSongPreviewWithoutCoverStartsAtFirstSlide(t *testing.T) {
	preview := NumberedSongPreview("Cântico novo", []string{"Primeira", "Segunda"})
	if !strings.Contains(preview, "SLIDE 01 · ESTROFE 1\nPrimeira") ||
		!strings.Contains(preview, "SLIDE 02 · ESTROFE 2\nSegunda") {
		t.Fatalf("numeração sem capa inesperada: %q", preview)
	}
}

func TestSaveSongDoesNotOverwriteInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "musicas.json")
	original := []byte("{ inválido")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveSong(path, "Música", "Estrofe"); err == nil {
		t.Fatal("esperava erro para JSON inválido")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("o JSON inválido foi alterado: %q", got)
	}
}

func TestSaveSongInfersCategoryAndCreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dados", "musicas.json")
	if err := SaveSong(path, "Salmo 23", "O Senhor é meu pastor."); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var songs Songs
	if err := json.Unmarshal(data, &songs); err != nil {
		t.Fatal(err)
	}
	if len(songs["Salmos"]["Salmo 23"]) != 1 {
		t.Fatalf("música não foi categorizada corretamente: %#v", songs)
	}
}
