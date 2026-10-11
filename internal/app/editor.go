package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"culto/internal/culto"
	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	title    string
	path     string
	original string
	err      error
}

func (m *model) openEditor(title string, stanzas []string) tea.Cmd {
	editor, err := findEditor()
	if err != nil {
		m.status = err.Error()
		return nil
	}

	file, err := os.CreateTemp("", "culto-*.md")
	if err != nil {
		m.status = fmt.Sprintf("não foi possível criar o arquivo temporário: %v", err)
		return nil
	}
	path := file.Name()
	original := songEditorText(title, stanzas)
	if _, err := file.WriteString(original); err != nil {
		file.Close()
		os.Remove(path)
		m.status = fmt.Sprintf("não foi possível preparar a letra para edição: %v", err)
		return nil
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		m.status = fmt.Sprintf("não foi possível fechar o arquivo temporário: %v", err)
		return nil
	}

	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{title: title, path: path, original: original, err: err}
	})
}

func findEditor() (string, error) {
	for _, name := range []string{"nvim", "vim"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("não encontrei o Vim nem o Neovim no PATH")
}

func (m *model) finishEditor(msg editorFinishedMsg) tea.Cmd {
	defer os.Remove(msg.path)
	if msg.err != nil {
		m.status = fmt.Sprintf("o editor foi encerrado com erro: %v", msg.err)
		return nil
	}
	content, err := os.ReadFile(msg.path)
	if err != nil {
		m.status = fmt.Sprintf("não foi possível ler a letra editada: %v", err)
		return nil
	}
	if string(content) == msg.original {
		m.editSong = ""
		m.popMenu()
		m.status = "Edição cancelada ou sem alterações."
		return nil
	}
	if err := culto.SaveSong(m.config.MusicFile, msg.title, songEditorLyrics(string(content), msg.title)); err != nil {
		m.status = fmt.Sprintf("erro ao salvar a música: %v", err)
		return nil
	}
	m.editSong = ""
	m.popMenu()
	m.status = fmt.Sprintf("Música %q salva.", msg.title)
	return nil
}

func songEditorText(title string, stanzas []string) string {
	var body string
	if len(stanzas) > 0 {
		body = strings.Join(stanzas, "\n\n")
	}
	return "# " + title + "\n\n" + body
}

func songEditorLyrics(content, title string) string {
	lines := strings.SplitN(strings.ReplaceAll(content, "\r\n", "\n"), "\n", 2)
	if len(lines) == 0 || lines[0] != "# "+title {
		return content
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.TrimLeft(lines[1], "\n")
}
