package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"culto/internal/culto"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMainMenuAndLiturgiaNavigation(t *testing.T) {
	m := New(culto.Config{}).(*model)
	if got := m.View(); !strings.Contains(got, "CULTO") || !strings.Contains(got, "Liturgia") {
		t.Fatalf("menu inicial inesperado: %s", got)
	}

	m.activate("liturgia")
	if m.screen != menuScreen || len(m.stack) != 1 {
		t.Fatal("liturgia não abriu como submenu")
	}
	m.activate("text:ORAÇÃO INICIAL")
	if m.screen != previewScreen || !strings.Contains(m.content, "ORAÇÃO INICIAL") {
		t.Fatal("item litúrgico não abriu a visualização")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != menuScreen || m.menu.Title != "Liturgia · Esc volta" {
		t.Fatal("Esc não voltou ao submenu de liturgia")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.menu.Title != "Culto" || len(m.stack) != 0 {
		t.Fatal("Esc não voltou ao menu principal")
	}
}

func TestSongEditorContentKeepsLyricsAndRemovesOnlyItsTitle(t *testing.T) {
	title := "Cântico de teste"
	original := songEditorText(title, []string{"Primeira estrofe", "Segunda estrofe"})
	want := "Primeira estrofe\n\nSegunda estrofe"
	if got := songEditorLyrics(original, title); got != want {
		t.Fatalf("letra extraída = %q, esperado %q", got, want)
	}

	content := "# Cântico de teste\n\n# Cabeçalho da estrofe\nletra"
	if got := songEditorLyrics(content, title); got != "# Cabeçalho da estrofe\nletra" {
		t.Fatalf("cabeçalho Markdown da letra foi removido: %q", got)
	}
}

func TestSongEditorStartsWithTitleAndEmptyLyrics(t *testing.T) {
	if got, want := songEditorText("Nova música", nil), "# Nova música\n\n"; got != want {
		t.Fatalf("conteúdo inicial = %q, esperado %q", got, want)
	}
}

func TestFinishEditorSavesLyricsAndReturnsToSongList(t *testing.T) {
	title := "Cântico existente"
	musicFile := filepath.Join(t.TempDir(), "musicas.json")
	if err := culto.SaveSong(musicFile, title, "Letra antiga"); err != nil {
		t.Fatal(err)
	}

	m := New(culto.Config{MusicFile: musicFile}).(*model)
	m.openSongs()
	m.openSong(title)

	original := songEditorText(title, []string{"Letra antiga"})
	edited := songEditorText(title, []string{"Primeira estrofe", "Segunda estrofe"})
	editFile := filepath.Join(t.TempDir(), "culto.md")
	if err := os.WriteFile(editFile, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	m.finishEditor(editorFinishedMsg{
		title: title, path: editFile, original: original,
	})

	if m.screen != menuScreen || m.menu.Title != "Músicas · Esc volta" {
		t.Fatal("após editar, a aplicação não voltou à lista de músicas")
	}
	songs, err := culto.LoadSongs(musicFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := songs["Cânticos Espirituais"][title]; len(got) != 2 ||
		got[0] != "Primeira estrofe" || got[1] != "Segunda estrofe" {
		t.Fatalf("letra editada não foi salva: %#v", got)
	}
	if _, err := os.Stat(editFile); !os.IsNotExist(err) {
		t.Fatal("arquivo temporário do editor não foi removido")
	}
}

func TestSongListHasAddAndBackActions(t *testing.T) {
	m := New(culto.Config{MusicFile: t.TempDir() + "/missing.json"}).(*model)
	m.openSongs()
	items := m.menu.Items()
	if len(items) < 2 {
		t.Fatal("lista de músicas sem ações básicas")
	}
	add, ok := items[0].(menuItem)
	if !ok || add.action != "song:add" {
		t.Fatalf("primeiro item não é adicionar: %#v", items[0])
	}
	back, ok := items[1].(menuItem)
	if !ok || back.action != "back" {
		t.Fatalf("segundo item não é voltar: %#v", items[1])
	}
}
