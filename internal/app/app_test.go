package app

import (
	"encoding/json"
	"fmt"
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
	if m.menu.Title != "" || len(m.stack) != 0 {
		t.Fatal("Esc não voltou ao menu principal")
	}
}

func TestMainMenuOmitsDuplicateActionsAndLiturgiaAgenda(t *testing.T) {
	agendaCount := 0
	for _, item := range mainItems() {
		entry := item.(menuItem)
		if entry.action == "clear" || entry.action == "quit" {
			t.Fatalf("ação redundante na lista inicial: %#v", entry)
		}
		if entry.action == "agenda" {
			agendaCount++
		}
	}
	if agendaCount != 1 {
		t.Fatalf("agenda deve aparecer uma vez no menu inicial; apareceu %d vezes", agendaCount)
	}
	m := New(culto.Config{}).(*model)
	m.activate("liturgia")
	for _, item := range m.menu.Items() {
		if entry := item.(menuItem); entry.action == "agenda" {
			t.Fatal("agenda aparece duplicada dentro de Liturgia")
		}
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

func TestSongCanBeRenamedFromPreview(t *testing.T) {
	musicFile := filepath.Join(t.TempDir(), "musicas.json")
	if err := culto.SaveSong(musicFile, "Título antigo", "Letra preservada"); err != nil {
		t.Fatal(err)
	}
	m := New(culto.Config{MusicFile: musicFile}).(*model)
	m.openSongs()
	m.openSong("Título antigo")
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m.input.SetValue("Título novo")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	songs, err := culto.LoadSongs(musicFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := songs["Cânticos Espirituais"]["Título novo"]; len(got) != 1 || got[0] != "Letra preservada" {
		t.Fatalf("música renomeada incorretamente: %#v", got)
	}
	if m.screen != menuScreen || m.menu.Index() < 2 {
		t.Fatal("após renomear, a lista não selecionou a música atualizada")
	}
}

func TestSongDeletionRequiresExplicitConfirmation(t *testing.T) {
	musicFile := filepath.Join(t.TempDir(), "musicas.json")
	if err := culto.SaveSong(musicFile, "Título a remover", "Letra"); err != nil {
		t.Fatal(err)
	}
	m := New(culto.Config{MusicFile: musicFile}).(*model)
	m.openSongs()
	m.openSong("Título a remover")
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.input.SetValue("não")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != previewScreen || m.status != "Remoção cancelada." {
		t.Fatal("confirmação incorreta não cancelou a remoção")
	}
	songs, err := culto.LoadSongs(musicFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(culto.SongTitles(songs)) != 1 {
		t.Fatal("música foi removida sem confirmação correta")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.input.SetValue("REMOVER")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	songs, err = culto.LoadSongs(musicFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(culto.SongTitles(songs)) != 0 {
		t.Fatalf("música ainda existe depois de confirmada a remoção: %#v", songs)
	}
	if m.screen != menuScreen {
		t.Fatal("após remover, a aplicação não voltou à lista de músicas")
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

func TestBibleReferenceCanReuseLastBookAndChapter(t *testing.T) {
	bibleFile := filepath.Join(t.TempDir(), "biblia.json")
	data, err := json.Marshal([]culto.BibleBook{{
		Name:     "Romanos",
		Abbrev:   "Rm",
		Chapters: [][]string{{"Versículo um.", "Versículo dois.", "Versículo três."}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bibleFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	m := New(culto.Config{BibleFile: bibleFile}).(*model)
	m.readBible("Romanos 1", false)
	if m.lastBibleBase != "Romanos 1" || !m.previewBible {
		t.Fatalf("contexto bíblico não foi armazenado: base=%q preview=%v", m.lastBibleBase, m.previewBible)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !strings.Contains(m.inputPrompt, "Romanos 1") {
		t.Fatalf("prompt não informa o capítulo atual: %q", m.inputPrompt)
	}
	m.input.SetValue("2")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != previewScreen || !strings.Contains(m.content, "Romanos 1:2") ||
		!strings.Contains(m.content, "2. Versículo dois.") {
		t.Fatalf("referência relativa não selecionou o versículo correto: %q", m.content)
	}
	if got := resolveBibleReference("10-12", "Romanos 1"); got != "Romanos 1:10-12" {
		t.Fatalf("intervalo relativo resolvido como %q", got)
	}
}

func TestPreviewErrorsCannotBeProjected(t *testing.T) {
	m := New(culto.Config{}).(*model)
	m.showError(fmt.Errorf("erro de teste"))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if m.previewProjectable || m.status != "Esta mensagem não pode ser projetada." {
		t.Fatalf("erro ficou projetável: projectable=%v status=%q", m.previewProjectable, m.status)
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

func TestPreviewArrowKeysMoveSentSlideWithoutLeavingPreview(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "xdotool-args")
	xdotool := filepath.Join(dir, "xdotool")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = search ]; then printf '4242\\n'; exit 0; fi\n" +
		"printf '%s\\n' \"$@\" >> \"$CULTO_TEST_XDOTOOL_ARGS\"\n"
	if err := os.WriteFile(xdotool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CULTO_TEST_XDOTOOL_ARGS", argsFile)

	m := New(culto.Config{}).(*model)
	m.showPreview("HINO\n\nESTROFE")
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.screen != previewScreen || m.status != "Slide anterior enviado ao sent." {
		t.Fatalf("atalho de slide anterior inesperado: tela=%v status=%q", m.screen, m.status)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.screen != previewScreen || m.status != "Slide seguinte enviado ao sent." {
		t.Fatalf("atalho de slide seguinte inesperado: tela=%v status=%q", m.screen, m.status)
	}

	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "key\n--window\n4242\n--clearmodifiers\nLeft\n" +
		"key\n--window\n4242\n--clearmodifiers\nRight\n"
	if string(got) != want {
		t.Fatalf("teclas enviadas ao sent:\n%s\nesperado:\n%s", got, want)
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
