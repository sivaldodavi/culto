package culto

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const slideTextLimit = 180

type BibleBook struct {
	Name     string     `json:"name"`
	Abbrev   string     `json:"abbrev"`
	Chapters [][]string `json:"chapters"`
}

type Verse struct {
	Book    string
	Chapter int
	Number  int
	Text    string
}

func LoadBible(path string) ([]BibleBook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lendo Bíblia %q: %w", path, err)
	}
	var books []BibleBook
	if err := json.Unmarshal(data, &books); err != nil {
		return nil, fmt.Errorf("interpretando Bíblia %q: %w", path, err)
	}
	if len(books) == 0 {
		return nil, fmt.Errorf("a Bíblia %q não contém livros", path)
	}
	return books, nil
}

func ParseReference(query string, books []BibleBook) ([]Verse, string, error) {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return nil, "", fmt.Errorf("informe uma referência bíblica")
	}
	bookName := strings.Join(fields, " ")
	location := ""
	if startsWithDigit(fields[len(fields)-1]) {
		location = fields[len(fields)-1]
		bookName = strings.Join(fields[:len(fields)-1], " ")
	}
	if bookName == "" {
		return nil, "", fmt.Errorf("informe o nome do livro")
	}
	book, err := findBook(bookName, books)
	if err != nil {
		return nil, "", err
	}
	chapter, firstVerse, lastVerse, err := parseLocation(location)
	if err != nil {
		return nil, "", err
	}
	if chapter > len(book.Chapters) {
		return nil, "", fmt.Errorf("capítulo %d não existe em %s", chapter, book.Name)
	}
	if firstVerse > 0 && lastVerse > len(book.Chapters[chapter-1]) {
		return nil, "", fmt.Errorf("versículo %d não existe em %s %d", lastVerse, book.Name, chapter)
	}

	var verses []Verse
	for chapterIndex, texts := range book.Chapters {
		chapterNumber := chapterIndex + 1
		if chapter != 0 && chapterNumber != chapter {
			continue
		}
		for verseIndex, text := range texts {
			verseNumber := verseIndex + 1
			if firstVerse > 0 && (verseNumber < firstVerse || verseNumber > lastVerse) {
				continue
			}
			verses = append(verses, Verse{book.Name, chapterNumber, verseNumber, text})
		}
	}
	if len(verses) == 0 {
		return nil, "", fmt.Errorf("referência %q não encontrada", query)
	}
	label := book.Name
	if location != "" {
		label += " " + location
	}
	return verses, label, nil
}

func findBook(query string, books []BibleBook) (BibleBook, error) {
	query = normalize(query)
	for _, book := range books {
		if normalize(book.Name) == query || normalize(book.Abbrev) == query {
			return book, nil
		}
	}
	seen := make(map[string]bool)
	var matches []BibleBook
	for _, book := range books {
		if strings.HasPrefix(normalize(book.Name), query) ||
			strings.HasPrefix(normalize(book.Abbrev), query) {
			if !seen[book.Name] {
				matches = append(matches, book)
				seen[book.Name] = true
			}
		}
	}
	switch len(matches) {
	case 0:
		return BibleBook{}, fmt.Errorf("livro %q não encontrado", query)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, book := range matches {
			names[i] = book.Name
		}
		return BibleBook{}, fmt.Errorf("referência ambígua %q: %s", query, strings.Join(names, ", "))
	}
}

func parseLocation(location string) (chapter, firstVerse, lastVerse int, err error) {
	if location == "" {
		return 0, 0, 0, nil
	}
	parts := strings.SplitN(location, ":", 2)
	chapter, err = strconv.Atoi(parts[0])
	if err != nil || chapter < 1 {
		return 0, 0, 0, fmt.Errorf("capítulo inválido em %q", location)
	}
	if len(parts) == 1 {
		return chapter, 0, 0, nil
	}
	verseParts := strings.SplitN(parts[1], "-", 2)
	firstVerse, err = strconv.Atoi(verseParts[0])
	if err != nil || firstVerse < 1 {
		return 0, 0, 0, fmt.Errorf("versículo inválido em %q", location)
	}
	lastVerse = firstVerse
	if len(verseParts) == 2 {
		lastVerse, err = strconv.Atoi(verseParts[1])
		if err != nil || lastVerse < firstVerse {
			return 0, 0, 0, fmt.Errorf("intervalo de versículos inválido em %q", location)
		}
	}
	return chapter, firstVerse, lastVerse, nil
}

func startsWithDigit(value string) bool {
	for _, r := range value {
		return unicode.IsDigit(r)
	}
	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func SearchBible(books []BibleBook, query string) ([]Verse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("informe um termo para buscar")
	}
	var results []Verse
	for _, book := range books {
		for chapterIndex, chapter := range book.Chapters {
			for verseIndex, text := range chapter {
				if strings.Contains(strings.ToLower(text), strings.ToLower(query)) {
					results = append(results, Verse{book.Name, chapterIndex + 1, verseIndex + 1, text})
				}
			}
		}
	}
	return results, nil
}

func RenderBible(verses []Verse, reference string, noCover, responsive bool) string {
	var slides []string
	if !noCover {
		slides = append(slides, strings.ToUpper(reference))
	}
	if responsive {
		for i, verse := range verses {
			speaker := "DIRIGENTE"
			if i%2 == 1 {
				speaker = "IGREJA"
			}
			slides = append(slides, fmt.Sprintf("%s %d (%s)\n%d. %s",
				verse.Book, verse.Chapter, speaker, verse.Number, verse.Text))
		}
		return strings.Join(slides, "\n\n")
	}
	var text []string
	for _, verse := range verses {
		text = append(text, fmt.Sprintf("%d. %s", verse.Number, verse.Text))
	}
	chunks := splitWords(strings.Join(text, " "), slideTextLimit)
	for i, chunk := range chunks {
		prefix, suffix := "", ""
		if i > 0 {
			prefix = "... "
		}
		if i < len(chunks)-1 {
			suffix = " ..."
		}
		slides = append(slides, reference+"\n"+prefix+chunk+suffix)
	}
	return strings.Join(slides, "\n\n")
}

func splitWords(text string, width int) []string {
	var lines []string
	var current strings.Builder
	for _, word := range strings.Fields(text) {
		if current.Len() > 0 && current.Len()+1+len(word) > width {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}
