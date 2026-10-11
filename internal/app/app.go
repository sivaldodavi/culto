package app

import (
	"fmt"
	"strconv"
	"strings"

	"culto/internal/culto"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen uint8

const (
	menuScreen screen = iota
	inputScreen
	previewScreen
)

type menuItem struct {
	title, description, action string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.description }
func (i menuItem) FilterValue() string { return i.title }

type menuState struct {
	title string
	items []list.Item
	index int
}

type model struct {
	config culto.Config
	screen screen
	menu   list.Model
	stack  []menuState

	input               textinput.Model
	inputPrompt         string
	inputAction         string
	preview             viewport.Model
	content             string
	editSong            string
	status              string
	inputReturn         screen
	lastBibleBase       string
	lastBibleResponsive bool
	previewBible        bool
	previewProjectable  bool
	jumpActive          bool
	jumpDigits          string
	width               int
	height              int
}

var heading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
var problem = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

func New(config culto.Config) tea.Model {
	m := &model{config: config, width: 80, height: 24}
	m.setMenu("", mainItems(), 0)
	m.preview = viewport.New(76, 17)
	return m
}

func mainItems() []list.Item {
	return []list.Item{
		menuItem{"Liturgia", "Orações, ceia e agenda", "liturgia"},
		menuItem{"Músicas", "Selecionar, projetar e editar letras", "songs"},
		menuItem{"Bíblia", "Leitura e leitura responsiva", "bible"},
		menuItem{"Buscar na Bíblia", "Pesquisar por palavra ou expressão", "search"},
		menuItem{"Agenda", "Projetar os horários semanais", "agenda"},
		menuItem{"PIX", "Abrir o QR code da igreja", "pix"},
	}
}

func menuWidth(width int) int {
	return max(1, min(64, width-8))
}

func (m *model) setMenu(title string, items []list.Item, index int) {
	m.menu = list.New(items, list.NewDefaultDelegate(), menuWidth(m.width), max(1, m.height-6))
	m.menu.Title = title
	m.menu.SetShowStatusBar(false)
	m.menu.SetShowHelp(false)
	m.menu.SetShowFilter(true)
	m.menu.SetFilteringEnabled(true)
	m.menu.Select(index)
	m.screen = menuScreen
}

func (m *model) pushMenu(title string, items []list.Item) {
	m.stack = append(m.stack, menuState{
		title: m.menu.Title,
		items: append([]list.Item(nil), m.menu.Items()...),
		index: m.menu.Index(),
	})
	m.setMenu(title, items, 0)
	m.status = ""
}

func (m *model) popMenu() {
	if len(m.stack) == 0 {
		m.setMenu("", mainItems(), 0)
		return
	}
	state := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	m.setMenu(state.title, state.items, state.index)
	m.status = ""
}

func (m *model) showPreview(content string) {
	m.showPreviewContent(content, true)
	m.previewBible = false
	m.editSong = ""
}

func (m *model) showPreviewContent(content string, projectable bool) {
	m.content = content
	m.preview.SetContent(culto.NumberedSlidesPreview(content))
	m.preview.GotoTop()
	m.screen = previewScreen
	m.previewProjectable = projectable
	m.status = ""
}

func (m *model) showSongPreview(title string, stanzas []string) {
	m.content = culto.RenderSong(title, stanzas)
	m.preview.SetContent(culto.NumberedSongPreview(title, stanzas))
	m.preview.GotoTop()
	m.screen = previewScreen
	m.previewProjectable = true
	m.previewBible = false
	m.status = ""
}

func (m *model) showError(err error) {
	m.showPreviewContent("Erro: "+err.Error(), false)
	m.previewBible = false
	m.editSong = ""
}

func (m *model) ask(prompt, action string) {
	m.inputReturn = m.screen
	m.input = textinput.New()
	m.input.Prompt = "> "
	m.input.CharLimit = 240
	m.input.Focus()
	m.inputPrompt, m.inputAction = prompt, action
	m.screen = inputScreen
	m.status = ""
}

func (m *model) askBibleReference(action string) {
	prompt := "Referência bíblica (ex.: Romanos 1:10)"
	if m.lastBibleBase != "" {
		prompt = fmt.Sprintf("Referência ou versículo de %s (ex.: 10 ou 10-12)", m.lastBibleBase)
	}
	m.ask(prompt, action)
}

func (m *model) cancelInput() {
	if m.inputReturn == previewScreen {
		m.screen = previewScreen
		m.status = ""
		return
	}
	m.popMenu()
}

func (m *model) Init() tea.Cmd { return textinput.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if finished, ok := msg.(editorFinishedMsg); ok {
		return m, m.finishEditor(finished)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menu.SetSize(menuWidth(msg.Width), max(1, msg.Height-6))
		m.preview.Width, m.preview.Height = max(1, msg.Width-4), max(1, msg.Height-8)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+l" {
			m.jumpActive = false
			m.jumpDigits = ""
			if err := culto.ClearPresentation(m.config); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Projeção limpa."
			}
			return m, nil
		}
	}
	switch m.screen {
	case menuScreen:
		return m.updateMenu(msg)
	case inputScreen:
		return m.updateInput(msg)
	case previewScreen:
		return m.updatePreview(msg)
	default:
		return m, nil
	}
}

func (m *model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			if len(m.stack) == 0 {
				return m, tea.Quit
			}
			m.popMenu()
			return m, nil
		case "esc":
			if len(m.stack) > 0 {
				m.popMenu()
			}
			return m, nil
		case "enter":
			item, ok := m.menu.SelectedItem().(menuItem)
			if ok {
				m.activate(item.action)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m *model) activate(action string) {
	switch action {
	case "liturgia":
		items := []list.Item{
			menuItem{"Oração inicial", "", "text:ORAÇÃO INICIAL"},
			menuItem{"Oração de contrição", "", "text:ORAÇÃO DE CONTRIÇÃO"},
			menuItem{"Oração pelas crianças", "", "text:ORAÇÃO PELAS CRIANÇAS"},
			menuItem{"Oração pela palavra", "", "text:ORAÇÃO PELA PALAVRA"},
			menuItem{"Ceia", "", "text:CEIA"},
			menuItem{"Ofertório", "", "text:OFERTÓRIO"},
			menuItem{"Oração final", "", "text:ORAÇÃO FINAL"},
			menuItem{"Boas-vindas", "", "text:BOAS-VINDAS"},
		}
		m.pushMenu("Liturgia · Esc volta", items)
	case "songs":
		m.openSongs()
	case "bible":
		m.pushMenu("Bíblia · Esc volta", []list.Item{
			menuItem{"Leitura", "Com capa de referência", "bible:read"},
			menuItem{"Leitura responsiva", "Alternar dirigente e igreja", "bible:responsive"},
		})
	case "search":
		m.pushAsk("Termo para pesquisar", "search")
	case "agenda":
		m.pushPreview(culto.AgendaText())
	case "pix":
		if err := culto.OpenPix(m.config.PixFile); err != nil {
			m.status = err.Error()
		} else {
			m.status = "Imagem PIX aberta."
		}
	default:
		switch {
		case strings.HasPrefix(action, "text:"):
			m.pushPreview(strings.TrimPrefix(action, "text:"))
		case strings.HasPrefix(action, "bible:"):
			m.stack = append(m.stack, menuState{
				title: m.menu.Title,
				items: append([]list.Item(nil), m.menu.Items()...),
				index: m.menu.Index(),
			})
			m.askBibleReference(strings.TrimPrefix(action, "bible:"))
		case action == "song:add":
			m.pushAsk("Título da nova música", "song:new")
		case strings.HasPrefix(action, "song:"):
			m.openSong(strings.TrimPrefix(action, "song:"))
		case action == "back":
			m.popMenu()
		}
	}
}

func (m *model) pushAsk(prompt, action string) {
	m.stack = append(m.stack, menuState{
		title: m.menu.Title,
		items: append([]list.Item(nil), m.menu.Items()...),
		index: m.menu.Index(),
	})
	m.ask(prompt, action)
}

func (m *model) pushPreview(content string) {
	m.stack = append(m.stack, menuState{
		title: m.menu.Title,
		items: append([]list.Item(nil), m.menu.Items()...),
		index: m.menu.Index(),
	})
	m.showPreview(content)
}

func (m *model) openSongs() {
	songs, err := culto.LoadSongs(m.config.MusicFile)
	if err != nil {
		m.showError(err)
		return
	}
	items := songItems(songs)
	m.pushMenu("Músicas · Esc volta", items)
}

func songItems(songs culto.Songs) []list.Item {
	items := []list.Item{
		menuItem{"＋ Adicionar música", "Criar uma letra", "song:add"},
		menuItem{"← Voltar", "", "back"},
	}
	for _, title := range culto.SongTitles(songs) {
		items = append(items, menuItem{title, "Enter abre; t renomeia e d remove na prévia", "song:" + title})
	}
	return items
}

func (m *model) refreshSongsMenu(selectedTitle string) {
	songs, err := culto.LoadSongs(m.config.MusicFile)
	if err != nil {
		m.status = err.Error()
		return
	}
	items := songItems(songs)
	index := 0
	for i, item := range items {
		if song, ok := item.(menuItem); ok && song.title == selectedTitle {
			index = i
			break
		}
	}
	if selectedTitle == "" && len(items) > 2 {
		index = 2
	}
	m.setMenu("Músicas · Esc volta", items, index)
}

func (m *model) openSong(title string) {
	songs, err := culto.LoadSongs(m.config.MusicFile)
	if err != nil {
		m.showError(err)
		return
	}
	for _, category := range songs {
		if stanzas, exists := category[title]; exists {
			m.stack = append(m.stack, menuState{
				title: m.menu.Title,
				items: append([]list.Item(nil), m.menu.Items()...),
				index: m.menu.Index(),
			})
			m.showSongPreview(title, stanzas)
			m.editSong = title
			return
		}
	}
	m.showError(fmt.Errorf("música %q não encontrada", title))
}

func (m *model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.cancelInput()
			return m, nil
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.status = "Informe um valor para continuar."
				return m, nil
			}
			m.status = ""
			switch m.inputAction {
			case "read", "responsive":
				m.readBible(value, m.inputAction == "responsive")
			case "search":
				m.searchBible(value)
			case "song:new":
				m.editSong = value
				return m, m.openEditor(value, nil)
			case "song:rename":
				oldTitle := m.editSong
				if err := culto.RenameSong(m.config.MusicFile, oldTitle, value); err != nil {
					m.status = err.Error()
					return m, nil
				}
				m.editSong = ""
				m.popMenu()
				m.refreshSongsMenu(value)
				m.status = fmt.Sprintf("Música renomeada para %q.", value)
			case "song:delete":
				if value != "REMOVER" {
					m.cancelInput()
					m.status = "Remoção cancelada."
					return m, nil
				}
				title := m.editSong
				if err := culto.DeleteSong(m.config.MusicFile, title); err != nil {
					m.status = err.Error()
					return m, nil
				}
				m.editSong = ""
				m.popMenu()
				m.refreshSongsMenu("")
				m.status = fmt.Sprintf("Música %q removida.", title)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) readBible(reference string, responsive bool) {
	books, err := culto.LoadBible(m.config.BibleFile)
	if err != nil {
		m.showError(err)
		return
	}
	reference = resolveBibleReference(reference, m.lastBibleBase)
	verses, label, err := culto.ParseReference(reference, books)
	if err != nil {
		m.showError(err)
		return
	}
	m.showPreview(culto.RenderBible(verses, label, false, responsive))
	m.previewBible = true
	m.lastBibleResponsive = responsive
	m.lastBibleBase = ""
	first := verses[0]
	sameChapter := true
	for _, verse := range verses[1:] {
		if verse.Book != first.Book || verse.Chapter != first.Chapter {
			sameChapter = false
			break
		}
	}
	if sameChapter {
		m.lastBibleBase = fmt.Sprintf("%s %d", first.Book, first.Chapter)
	}
}

func resolveBibleReference(reference, base string) string {
	reference = strings.TrimSpace(reference)
	relative := strings.TrimPrefix(reference, ":")
	if base != "" && isVerseRange(relative) {
		return base + ":" + relative
	}
	return reference
}

func isVerseRange(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 1 {
			return false
		}
	}
	return true
}

func (m *model) searchBible(query string) {
	books, err := culto.LoadBible(m.config.BibleFile)
	if err != nil {
		m.showError(err)
		return
	}
	results, err := culto.SearchBible(books, query)
	if err != nil {
		m.showError(err)
		return
	}
	if len(results) == 0 {
		m.showPreview(fmt.Sprintf("Nenhuma ocorrência de %q.", query))
		return
	}
	lines := []string{fmt.Sprintf("%d ocorrência(s) de %q", len(results), query)}
	for _, verse := range results {
		lines = append(lines, fmt.Sprintf("%s %d:%d\n%s",
			verse.Book, verse.Chapter, verse.Number, verse.Text))
	}
	m.showPreview(strings.Join(lines, "\n\n"))
}

func (m *model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.jumpActive {
			return m.updateSlideJump(key)
		}
		switch key.String() {
		case "esc", "q":
			m.editSong = ""
			m.popMenu()
			return m, nil
		case "g":
			m.jumpActive = true
			m.jumpDigits = ""
			m.status = ""
			return m, nil
		case "left", "right":
			direction := 1
			directionName := "seguinte"
			if key.String() == "left" {
				direction = -1
				directionName = "anterior"
			}
			if err := culto.MoveSlide(direction); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Slide " + directionName + " enviado ao sent."
			}
			return m, nil
		case "v":
			if m.previewBible {
				action := "read"
				if m.lastBibleResponsive {
					action = "responsive"
				}
				m.askBibleReference(action)
			}
			return m, nil
		case "p":
			if !m.previewProjectable {
				m.status = "Esta mensagem não pode ser projetada."
				return m, nil
			}
			if err := culto.Present(m.config, m.content); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Conteúdo enviado ao sent."
			}
			return m, nil
		case "e":
			if m.editSong != "" {
				songs, err := culto.LoadSongs(m.config.MusicFile)
				if err != nil {
					m.status = err.Error()
					return m, nil
				}
				for _, category := range songs {
					if stanzas, ok := category[m.editSong]; ok {
						return m, m.openEditor(m.editSong, stanzas)
					}
				}
			}
		case "t":
			if m.editSong != "" {
				m.ask("Novo título para "+m.editSong, "song:rename")
			}
		case "d":
			if m.editSong != "" {
				m.ask("Digite REMOVER para excluir "+m.editSong, "song:delete")
			}
		}
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m *model) updateSlideJump(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	value := key.String()
	switch value {
	case "esc":
		m.jumpActive = false
		m.jumpDigits = ""
		m.status = ""
	case "enter":
		if m.jumpDigits == "" {
			m.status = "Digite o número do slide."
			return m, nil
		}
		slide, err := strconv.Atoi(m.jumpDigits)
		if err != nil {
			m.status = "Número de slide inválido."
		} else if err := culto.JumpToSlide(slide); err != nil {
			m.status = err.Error()
		} else {
			m.status = fmt.Sprintf("Enviado ao sent: slide %d.", slide)
			m.jumpActive = false
			m.jumpDigits = ""
		}
	case "backspace", "ctrl+h":
		if len(m.jumpDigits) > 0 {
			m.jumpDigits = m.jumpDigits[:len(m.jumpDigits)-1]
		}
	default:
		if len(value) == 1 && value[0] >= '0' && value[0] <= '9' && len(m.jumpDigits) < 6 {
			m.jumpDigits += value
			m.status = ""
		}
	}
	return m, nil
}

func (m *model) View() string {
	frameWidth := max(1, m.width-4)
	header := lipgloss.PlaceHorizontal(frameWidth, lipgloss.Center,
		heading.Render("CULTO")+"  "+muted.Render("liturgia e projeção"))
	var body string
	switch m.screen {
	case menuScreen:
		body = lipgloss.PlaceHorizontal(frameWidth, lipgloss.Center, m.menu.View())
	case inputScreen:
		body = "\n" + m.inputPrompt + "\n\n" + m.input.View()
	case previewScreen:
		body = m.preview.View()
	}
	if m.jumpActive {
		body += "\n\n" + muted.Render("Ir para slide: "+m.jumpDigits+"▏")
	}
	if m.status != "" {
		body += "\n" + problem.Width(frameWidth).Align(lipgloss.Center).Render(m.status)
	}
	if m.screen == inputScreen || m.screen == menuScreen {
		body = lipgloss.PlaceHorizontal(frameWidth, lipgloss.Center, body)
	}
	help := muted.Width(frameWidth).Align(lipgloss.Center).Render(m.help())
	return lipgloss.NewStyle().Padding(1, 2).Render(header + "\n\n" + body + "\n\n" + help)
}

func (m *model) help() string {
	global := " · Ctrl+L limpar projeção"
	switch m.screen {
	case menuScreen:
		return "↑/↓ navegar · / filtrar · Enter abrir · Esc voltar · q sair" + global
	case inputScreen:
		return "Enter confirmar · Esc cancelar" + global
	case previewScreen:
		if m.jumpActive {
			return "Digite o número · Enter ir · Backspace apagar · Esc cancelar" + global
		}
		actions := "←/→ slide anterior/seguinte · ↑/↓ rolar · g ir para slide"
		if m.previewProjectable {
			actions += " · p projetar"
		}
		if m.previewBible {
			actions += " · v ajustar referência"
		}
		if m.editSong != "" {
			actions += " · e editar letra · t renomear · d remover"
		}
		return actions + " · Esc voltar" + global
	default:
		return ""
	}
}
