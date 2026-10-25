package app

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"culto/internal/culto"
	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	title         string
	path          string
	livePath      string
	original      string
	projectionErr error
	err           error
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

	liveFile, err := os.CreateTemp("", "culto-live-*.md")
	if err != nil {
		os.Remove(path)
		m.status = fmt.Sprintf("não foi possível criar o arquivo de projeção ao vivo: %v", err)
		return nil
	}
	livePath := liveFile.Name()
	if _, err := liveFile.WriteString(original); err != nil {
		liveFile.Close()
		os.Remove(path)
		os.Remove(livePath)
		m.status = fmt.Sprintf("não foi possível preparar a projeção ao vivo: %v", err)
		return nil
	}
	if err := liveFile.Close(); err != nil {
		os.Remove(path)
		os.Remove(livePath)
		m.status = fmt.Sprintf("não foi possível fechar o arquivo de projeção ao vivo: %v", err)
		return nil
	}

	autocmd := fmt.Sprintf("autocmd TextChanged,TextChangedI <buffer> call writefile(getline(1, '$'), %s)", strconv.Quote(livePath))
	stopProjection := watchSongProjection(m.config, livePath, title, original)
	cmd := exec.Command(editor, "-c", autocmd, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{
			title: title, path: path, livePath: livePath, original: original,
			projectionErr: stopProjection(), err: err,
		}
	})
}

func watchSongProjection(config culto.Config, path, title, original string) func() error {
	stop := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		lastContent := original
		pendingContent := ""
		pending := false
		var changedAt time.Time
		for {
			select {
			case <-stop:
				content, err := os.ReadFile(path)
				if err != nil {
					done <- fmt.Errorf("lendo a letra para a projeção ao vivo: %w", err)
					return
				}
				if contentChanged(content, lastContent) {
					done <- projectSongEditorText(config, title, string(content))
					return
				}
				if pending && contentChanged([]byte(pendingContent), lastContent) {
					done <- projectSongEditorText(config, title, pendingContent)
					return
				}
				done <- nil
				return
			case <-ticker.C:
				content, err := os.ReadFile(path)
				if err != nil {
					done <- fmt.Errorf("lendo a letra para a projeção ao vivo: %w", err)
					return
				}
				switch {
				case !contentChanged(content, lastContent):
					pending = false
				case !pending || string(content) != pendingContent:
					pendingContent = string(content)
					changedAt = time.Now()
					pending = true
				case time.Since(changedAt) >= 150*time.Millisecond:
					if err := projectSongEditorText(config, title, pendingContent); err != nil {
						done <- err
						return
					}
					lastContent = pendingContent
					pending = false
				}
			}
		}
	}()

	var stopped bool
	return func() error {
		if !stopped {
			close(stop)
			stopped = true
		}
		return <-done
	}
}

func contentChanged(content []byte, previous string) bool {
	return string(content) != previous
}

func projectSongEditorText(config culto.Config, title, content string) error {
	lyrics := songEditorLyrics(content, title)
	var stanzas []string
	for _, stanza := range strings.Split(strings.ReplaceAll(lyrics, "\r\n", "\n"), "\n\n") {
		if stanza = strings.TrimSpace(stanza); stanza != "" {
			stanzas = append(stanzas, stanza)
		}
	}
	return culto.Present(config, culto.RenderSong(title, stanzas))
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
	defer os.Remove(msg.livePath)
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
		liveContent, err := os.ReadFile(msg.livePath)
		if err != nil {
			m.status = fmt.Sprintf("Edição cancelada, mas não foi possível verificar a projeção ao vivo: %v", err)
			return nil
		}
		if string(liveContent) != msg.original {
			if err := projectSongEditorText(m.config, msg.title, msg.original); err != nil {
				m.status = fmt.Sprintf("Edição descartada, mas não foi possível restaurar a projeção: %v", err)
				return nil
			}
			m.status = "Edição descartada; projeção restaurada."
			return nil
		}
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
	if msg.projectionErr != nil {
		m.status += fmt.Sprintf(" Erro na projeção ao vivo: %v", msg.projectionErr)
	}
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
