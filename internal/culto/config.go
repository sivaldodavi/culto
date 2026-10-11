package culto

import (
	"os"
	"path/filepath"
)

type Config struct {
	BibleFile string
	MusicFile string
	PixFile   string
	SentFile  string
}

func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	dir := filepath.Join(home, "Dufs", "Igreja", "Projeção")
	if override := os.Getenv("CULTO_DIR"); override != "" {
		dir = override
	}
	return Config{
		BibleFile: envOr("CULTO_BIBLE_FILE", filepath.Join(dir, "biblia.json")),
		MusicFile: envOr("CULTO_MUSIC_FILE", filepath.Join(dir, "musicas.json")),
		PixFile:   envOr("CULTO_PIX_FILE", filepath.Join(dir, "pix-igreja.png")),
		SentFile:  envOr("CULTO_SENT_FILE", filepath.Join(os.TempDir(), "sent_presentation")),
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
