package culto

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const slideWidth = 40

func Present(config Config, content string) error {
	if err := writeAtomic(config.SentFile, []byte(formatSlides(content))); err != nil {
		return fmt.Errorf("preparando apresentação: %w", err)
	}
	return refreshOrStart(config.SentFile, true)
}

func ClearPresentation(config Config) error {
	if err := writeAtomic(config.SentFile, []byte(" \n")); err != nil {
		return fmt.Errorf("limpando apresentação: %w", err)
	}
	return refreshOrStart(config.SentFile, false)
}

func JumpToSlide(slide int) error {
	if slide < 1 {
		return fmt.Errorf("o número do slide deve ser maior que zero")
	}
	xdotool, err := exec.LookPath("xdotool")
	if err != nil {
		return fmt.Errorf("xdotool não foi encontrado")
	}
	window, err := findSentWindow(xdotool)
	if err != nil {
		return err
	}
	args := []string{"key", "--window", window, "--clearmodifiers", "g"}
	for _, digit := range strconv.Itoa(slide) {
		args = append(args, string(digit))
	}
	args = append(args, "Return")
	if err := exec.Command(xdotool, args...).Run(); err != nil {
		return fmt.Errorf("não foi possível enviar o salto para o slide %d: %w", slide, err)
	}
	return nil
}

func MoveSlide(direction int) error {
	key := ""
	switch direction {
	case -1:
		key = "Left"
	case 1:
		key = "Right"
	default:
		return fmt.Errorf("a direção do slide deve ser -1 ou 1")
	}
	xdotool, err := exec.LookPath("xdotool")
	if err != nil {
		return fmt.Errorf("xdotool não foi encontrado")
	}
	window, err := findSentWindow(xdotool)
	if err != nil {
		return err
	}
	if err := exec.Command(xdotool, "key", "--window", window, "--clearmodifiers", key).Run(); err != nil {
		return fmt.Errorf("não foi possível avançar ou voltar o slide: %w", err)
	}
	return nil
}

func findSentWindow(xdotool string) (string, error) {
	output, err := exec.Command(xdotool, "search", "--onlyvisible", "--class", "sent").Output()
	if err != nil {
		return "", fmt.Errorf("não encontrei uma janela visível do sent")
	}
	windows := strings.Fields(string(output))
	if len(windows) == 0 {
		return "", fmt.Errorf("não encontrei uma janela visível do sent")
	}
	window := windows[0]
	if _, err := strconv.ParseUint(window, 10, 64); err != nil {
		return "", fmt.Errorf("o identificador da janela do sent é inválido")
	}
	return window, nil
}

func formatSlides(content string) string {
	paragraphs := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n")
	var slides []string
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph != "" {
			var lines []string
			for _, line := range strings.Split(paragraph, "\n") {
				lines = append(lines, wrap(strings.ToUpper(strings.TrimSpace(line)), slideWidth)...)
			}
			slides = append(slides, strings.Join(lines, "\n"))
		}
	}
	return strings.Join(slides, "\n\n") + "\n"
}

func NumberedSlidesPreview(content string) string {
	formatted := strings.TrimSpace(formatSlides(content))
	if formatted == "" {
		return ""
	}
	slides := strings.Split(formatted, "\n\n")
	numbered := make([]string, len(slides))
	for i, slide := range slides {
		numbered[i] = fmt.Sprintf("SLIDE %02d\n%s", i+1, slide)
	}
	return strings.Join(numbered, "\n\n")
}

func wrap(text string, width int) []string {
	var lines []string
	var line strings.Builder
	for _, word := range strings.Fields(text) {
		if line.Len() > 0 && line.Len()+1+len(word) > width {
			lines = append(lines, line.String())
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteByte(' ')
		}
		line.WriteString(word)
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

func refreshOrStart(path string, startIfMissing bool) error {
	xdotool, xdotoolErr := exec.LookPath("xdotool")
	if xdotoolErr == nil {
		output, err := exec.Command(xdotool, "search", "--onlyvisible", "--class", "sent").Output()
		if err == nil {
			windows := strings.Fields(string(output))
			for _, window := range windows {
				if _, parseErr := strconv.ParseUint(window, 10, 64); parseErr != nil {
					continue
				}
				if err := exec.Command(xdotool, "key", "--window", window, "r").Run(); err == nil {
					return nil
				}
			}
		}
	}
	pgrep, err := exec.LookPath("pgrep")
	if err != nil {
		if !startIfMissing {
			return nil
		}
		return fmt.Errorf("pgrep não foi encontrado; não é possível verificar se sent já está aberto")
	}
	running := exec.Command(pgrep, "-x", "sent").Run() == nil
	if running {
		if xdotoolErr != nil {
			return fmt.Errorf("sent já está aberto, mas xdotool não foi encontrado")
		}
		return fmt.Errorf("sent já está aberto, mas não foi possível atualizar a apresentação")
	}
	if !startIfMissing {
		return nil
	}
	sent, err := exec.LookPath("sent")
	if err != nil {
		return fmt.Errorf("sent não está aberto e não foi encontrado no PATH")
	}
	return startSent(sent, path)
}

func startSent(sent, slideFile string) error {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return startDetached(sent, slideFile)
	}

	hyprctl, err := exec.LookPath("hyprctl")
	if err != nil {
		return fmt.Errorf("hyprctl não foi encontrado; não é possível iniciar sent no monitor HDMI-A-1")
	}
	output, err := exec.Command(hyprctl, "monitors", "-j").Output()
	if err != nil {
		return fmt.Errorf("não foi possível consultar os monitores do Hyprland: %w", err)
	}
	var monitors []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &monitors); err != nil {
		return fmt.Errorf("resposta inválida ao consultar monitores do Hyprland: %w", err)
	}
	found := false
	for _, monitor := range monitors {
		if monitor.Name == "HDMI-A-1" {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("o monitor HDMI-A-1 não está conectado; sent não foi iniciado")
	}

	command := shellQuote(sent) + " " + shellQuote(slideFile)
	dispatch := fmt.Sprintf("hl.dsp.exec_cmd(%s, { monitor = %s })",
		strconv.Quote(command), strconv.Quote("HDMI-A-1"))
	output, err = exec.Command(hyprctl, "dispatch", dispatch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("não foi possível iniciar sent no monitor HDMI-A-1: %w: %s",
			err, strings.TrimSpace(string(output)))
	}
	if response := strings.TrimSpace(string(output)); response != "ok" {
		return fmt.Errorf("Hyprland não confirmou a abertura do sent no monitor HDMI-A-1: %s", response)
	}
	return nil
}

func startDetached(sent, slideFile string) error {
	cmd := exec.Command(sent, slideFile)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("iniciando sent: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("liberando processo do sent: %w", err)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func OpenPix(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("verificando imagem PIX %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("o caminho PIX %q não é um arquivo regular", path)
	}
	opener, err := exec.LookPath("xdg-open")
	if err != nil {
		return fmt.Errorf("xdg-open não foi encontrado")
	}
	cmd := exec.Command(opener, path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("abrindo imagem PIX: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("liberando processo do visualizador: %w", err)
	}
	return nil
}

func AgendaText() string {
	return strings.Join([]string{
		"Terça (19:30)   - Culto de oração",
		"Quinta (19:30)  - Culto de doutrina",
		"Domingo (08:30) - EBD",
		"Domingo (18:00) - Culto solene",
	}, "\n")
}
