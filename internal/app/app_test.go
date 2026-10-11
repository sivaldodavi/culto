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

func TestSlideJumpStaysOnPreviewAndSendsNumberToSent(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "xdotool-args")
	xdotool := filepath.Join(dir, "xdotool")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = search ]; then printf '4242\\n'; exit 0; fi\n" +
		"printf '%s\\n' \"$@\" > \"$CULTO_TEST_XDOTOOL_ARGS\"\n"
	if err := os.WriteFile(xdotool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CULTO_TEST_XDOTOOL_ARGS", argsFile)

	m := New(culto.Config{}).(*model)
	m.showPreview("HINO\n\nESTROFE")
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if m.screen != previewScreen || !m.jumpActive || m.jumpDigits != "12" {
		t.Fatal("entrada do número trocou a tela de projeção")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != previewScreen || m.jumpActive {
		t.Fatal("após o salto, a tela de projeção não foi mantida")
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "key\n--window\n4242\n--clearmodifiers\ng\n1\n2\nReturn\n"
	if string(got) != want {
		t.Fatalf("sequência enviada ao sent:\n%s\nesperado:\n%s", got, want)
	}
}

func TestSlideJumpCanBeCancelledWithoutLeavingPreview(t *testing.T) {
	m := New(culto.Config{}).(*model)
	m.showPreview("SLIDE")
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != previewScreen || m.jumpActive || m.jumpDigits != "" {
		t.Fatal("cancelar o salto alterou a tela ou manteve o número digitado")
	}
}

func TestCtrlLClearsProjectionWithoutLeavingCurrentScreen(t *testing.T) {
	for _, screenName := range []string{"menu", "input", "preview"} {
		t.Run(screenName, func(t *testing.T) {
			dir := t.TempDir()
			xdotool := filepath.Join(dir, "xdotool")
			script := "#!/bin/sh\n" +
				"if [ \"$1\" = search ]; then printf '4242\\n'; exit 0; fi\n" +
				"exit 0\n"
			if err := os.WriteFile(xdotool, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)

			sentFile := filepath.Join(dir, "sent-slides")
			if err := os.WriteFile(sentFile, []byte("SLIDE 1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			m := New(culto.Config{SentFile: sentFile}).(*model)
			switch screenName {
			case "input":
				m.ask("Buscar", "search")
				m.input.SetValue("termo")
			case "preview":
				m.showPreview("Conteúdo atual")
			}
			beforeScreen := m.screen

			m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})

			if m.screen != beforeScreen {
				t.Fatalf("atalho mudou de tela: %v -> %v", beforeScreen, m.screen)
			}
			if m.status != "Projeção limpa." {
				t.Fatalf("status inesperado: %q", m.status)
			}
			got, err := os.ReadFile(sentFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != " \n" {
				t.Fatalf("arquivo da projeção não foi limpo: %q", got)
			}
			if screenName == "input" && m.input.Value() != "termo" {
				t.Fatalf("texto do campo foi perdido: %q", m.input.Value())
			}
			if screenName == "preview" && m.content != "Conteúdo atual" {
				t.Fatalf("conteúdo da tela foi perdido: %q", m.content)
			}
		})
	}
}
