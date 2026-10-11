package culto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Songs map[string]map[string][]string

var songCategories = []string{"Hinos", "Salmos", "Cânticos Espirituais"}
var hymnTitle = regexp.MustCompile(`^(\d+(?:-[A-Za-z])?)\s*-\s*.+$`)

func LoadSongs(path string) (Songs, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return emptySongs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo músicas %q: %w", path, err)
	}
	var songs Songs
	if err := json.Unmarshal(data, &songs); err != nil {
		return nil, fmt.Errorf("interpretando músicas %q: %w", path, err)
	}
	if songs == nil {
		return nil, fmt.Errorf("o arquivo %q não contém um objeto JSON", path)
	}
	for _, category := range songCategories {
		if songs[category] == nil {
			songs[category] = make(map[string][]string)
		}
	}
	return songs, nil
}

func emptySongs() Songs {
	songs := make(Songs, len(songCategories))
	for _, category := range songCategories {
		songs[category] = make(map[string][]string)
	}
	return songs
}

func SongTitles(songs Songs) []string {
	var titles []string
	for _, category := range songs {
		for title := range category {
			titles = append(titles, title)
		}
	}
	sort.Strings(titles)
	return titles
}

func RenderSong(title string, stanzas []string) string {
	return strings.Join(songSlides(title, stanzas), "\n\n")
}

func NumberedSongPreview(title string, stanzas []string) string {
	slides := songSlides(title, stanzas)
	hasCover := len(slides) > len(stanzas)
	numbered := make([]string, 0, len(slides))
	for i, slide := range slides {
		label := fmt.Sprintf("SLIDE %02d · ESTROFE %d", i+1, i+1)
		if hasCover && i == 0 {
			label = fmt.Sprintf("SLIDE %02d · CAPA", i+1)
		} else if hasCover {
			label = fmt.Sprintf("SLIDE %02d · ESTROFE %d", i+1, i)
		}
		numbered = append(numbered, label+"\n"+slide)
	}
	return strings.Join(numbered, "\n\n")
}

func songSlides(title string, stanzas []string) []string {
	var slides []string
	if match := hymnTitle.FindStringSubmatch(title); match != nil {
		slides = append(slides, "HINO "+match[1])
	} else if strings.Contains(strings.ToLower(title), "salmo") {
		slides = append(slides, title)
	}
	slides = append(slides, stanzas...)
	return slides
}

func SaveSong(path, title, content string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("o título não pode ficar vazio")
	}
	var stanzas []string
	for _, stanza := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n") {
		if stanza = strings.TrimSpace(stanza); stanza != "" {
			stanzas = append(stanzas, stanza)
		}
	}
	if len(stanzas) == 0 {
		return fmt.Errorf("a letra está vazia; nada foi salvo")
	}
	songs, err := LoadSongs(path)
	if err != nil {
		return err
	}
	category := findCategory(songs, title)
	if category == "" {
		category = inferCategory(title)
	}
	songs[category][title] = stanzas
	data, err := json.MarshalIndent(songs, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando músicas: %w", err)
	}
	return writeAtomic(path, append(data, '\n'))
}

func findCategory(songs Songs, title string) string {
	for category, titles := range songs {
		if _, exists := titles[title]; exists {
			return category
		}
	}
	return ""
}

func inferCategory(title string) string {
	switch {
	case hymnTitle.MatchString(title):
		return "Hinos"
	case strings.Contains(strings.ToLower(title), "salmo"):
		return "Salmos"
	default:
		return "Cânticos Espirituais"
	}
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("criando diretório %q: %w", dir, err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("verificando arquivo %q: %w", path, err)
	}
	file, err := os.CreateTemp(dir, ".culto-*.tmp")
	if err != nil {
		return fmt.Errorf("criando arquivo temporário: %w", err)
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return fmt.Errorf("ajustando permissões temporárias: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("gravando arquivo temporário: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sincronizando arquivo temporário: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("fechando arquivo temporário: %w", err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("substituindo arquivo %q: %w", path, err)
	}
	return nil
}
