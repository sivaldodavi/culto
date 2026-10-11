package app

import (
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

func TestCreateSongThroughEditor(t *testing.T) {
	musicFile := t.TempDir() + "/musicas.json"
	m := New(culto.Config{MusicFile: musicFile}).(*model)
	m.activate("songs")
	m.activate("song:add")
	if m.screen != inputScreen {
		t.Fatal("formulário de título não abriu")
	}
	m.input.SetValue("Hino novo")
	m.updateInput(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != editorScreen {
		t.Fatal("editor da música não abriu")
	}
	m.textarea.SetValue("Primeira estrofe\n\nSegunda estrofe")
	m.updateEditor(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.screen != menuScreen || m.menu.Title != "Músicas · Esc volta" {
		t.Fatal("após salvar, a aplicação não voltou à lista de músicas")
	}
	songs, err := culto.LoadSongs(musicFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs["Cânticos Espirituais"]["Hino novo"]) != 2 {
		t.Fatalf("estrofes não salvas: %#v", songs)
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
