#!/usr/bin/env bash

CULTO="$HOME/Dufs/Igreja/Projeção"

# ==============================================================================
# Configuração
# ==============================================================================

BIBLE_FILE="$CULTO/biblia.json"
MUSICAS_FILE="$CULTO/musicas.json"
PIX_FILE="$CULTO/pix-igreja.png"
AGENDA_FILE="$CULTO/agenda.jpeg"
SENT_FILE="/tmp/sent_presentation"

: "${EDITOR:=vim}"

LITURGIA_FIXA=(
    "T: oração inicial"
    "T: oração de contrição"
    "T: oração pelas crianças"
    "T: oração pela palavra"
    "T: ceia"
    "T: ofertório"
    "T: oração final"
    "T: boas vindas"
    "AG: agenda"
    "C: limpar tela"
)

AGENDA_TEXTO="Terça (19:30)   - Culto de oração
Quinta (19:30)  - Culto de doutrina
Domingo (08:30) - EBD
Domingo (18:00) - Culto solene"

# ==============================================================================
# Funções Auxiliares e Garantias
# ==============================================================================

_ensure_bible() {
    if [ ! -f "$BIBLE_FILE" ]; then
        echo "ERRO: Arquivo de bíblia não encontrado em: $BIBLE_FILE" >&2
        return 1
    fi
}

_ensure_musicas() {
    if [ ! -f "$MUSICAS_FILE" ]; then
        mkdir -p "$(dirname "$MUSICAS_FILE")"
        printf '{\n  "Hinos": {},\n  "Salmos": {},\n  "Cânticos Espirituais": {}\n}\n' > "$MUSICAS_FILE"
    fi
}

# ==============================================================================
# Integração ao 'sent' (Wayland / Hyprland via Xwayland)
# ==============================================================================

format_sent() {
    awk '{ print toupper($0) }' \
        | fold -s -w 40 \
        | sed 's/[[:space:]]*$//' \
        > "$SENT_FILE"

    if pgrep -x "sent" >/dev/null; then
        DISPLAY="${DISPLAY:-:0}" xdotool search --class sent key r 2>/dev/null || {
            pkill -x sent
            nohup sent "$SENT_FILE" >/dev/null 2>&1 &
        }
    else
        nohup sent "$SENT_FILE" >/dev/null 2>&1 &
    fi
}

clear_sent() {
    printf ' ' > "$SENT_FILE"
    if pgrep -x "sent" >/dev/null; then
        DISPLAY="${DISPLAY:-:0}" xdotool search --class sent key r 2>/dev/null || {
            pkill -x sent
            nohup sent "$SENT_FILE" >/dev/null 2>&1 &
        }
    fi
}

# ==============================================================================
# Bíblia Local
# ==============================================================================

_bible_python() {
    BIBLE_FILE="$BIBLE_FILE" python3 - "$@" <<'PYEOF'
import os, re, sys, signal, json

signal.signal(signal.SIGPIPE, signal.SIG_DFL)

BIBLE_FILE  = os.environ["BIBLE_FILE"]
SLIDE_WIDTH = 180

USE_COLOR = sys.stdout.isatty()
def _c(code, t): return f"\033[{code}m{t}\033[0m" if USE_COLOR else t
def azul(t):     return _c("1;34", t)
def amarelo(t):  return _c("1;33", t)
def verde(t):    return _c("1;32", t)
def vermelho(t): return _c("1;31", t)

def load_bible():
    verses = []
    with open(BIBLE_FILE, encoding="utf-8") as f:
        data = json.load(f)
        for book_obj in data:
            book_name = book_obj.get("name", "")
            book_abbrev = book_obj.get("abbrev", "")

            for cap_idx, chapter in enumerate(book_obj.get("chapters", [])):
                cap_num = cap_idx + 1

                for vers_idx, text in enumerate(chapter):
                    vers_num = vers_idx + 1

                    verses.append({
                        "book": book_name,
                        "abbrev": book_abbrev,
                        "chapter": cap_num,
                        "verse": vers_num,
                        "text": text
                    })
    return verses

def parse_ref(query, verses):
    query = query.strip()
    parts = query.rpartition(" ")
    book, num = parts[0].strip(), parts[2].strip()
    if not book:
        book, num = num, ""
    try:
        if ":" in num:
            cap_str, vrs_part = num.split(":", 1)
            cap_num = int(cap_str)
            if "-" in vrs_part:
                v1, v2 = (int(x) for x in vrs_part.split("-", 1))
            else:
                v1 = v2 = int(vrs_part)
            return [v for v in verses
                    if (v["book"].lower().startswith(book.lower()) or v["abbrev"].lower() == book.lower())
                    and v["chapter"] == cap_num
                    and v1 <= v["verse"] <= v2]
        else:
            cap_num = int(num) if num.isdigit() else 0
            return [v for v in verses
                    if (v["book"].lower().startswith(book.lower()) or v["abbrev"].lower() == book.lower())
                    and (cap_num == 0 or v["chapter"] == cap_num)]
    except ValueError:
        return []

def _chunks(text, width):
    words, chunks, current = text.split(), [], ""
    for word in words:
        if len(current) + len(word) + 1 > width:
            chunks.append(current.rstrip())
            current = word + " "
        else:
            current += word + " "
    if current.strip():
        chunks.append(current.rstrip())
    return chunks

def slides_read(rows, ref_label, no_cover, resp):
    out = []
    if not no_cover:
        out += [ref_label.upper(), ""]
    if resp:
        for i, row in enumerate(rows):
            label = "DIRIGENTE" if i % 2 == 0 else "IGREJA"
            header = f"{row['book']} {row['chapter']} ({label})"
            out += [f"{header}\n{row['verse']}. {row['text']}", ""]
    else:
        full_text = " ".join(f"{r['verse']}. {r['text']}" for r in rows)
        chunks = _chunks(full_text, SLIDE_WIDTH)
        for i, chunk in enumerate(chunks):
            pre = "... " if i > 0 else ""
            pos = " ..." if i < len(chunks) - 1 else ""
            out += [f"{ref_label}\n{pre}{chunk}{pos}", ""]
    return "\n".join(out)

def terminal_read(rows, ref_label):
    print(azul(ref_label))
    print()
    for r in rows:
        num = verde(f"{r['verse']:3}.")
        print(f"  {num} {r['text']}")
    print()

cmd  = sys.argv[1]
rest = sys.argv[2:]

if cmd == "read":
    no_cover = "--no-cover" in rest
    resp     = "--resp"     in rest
    query    = " ".join(a for a in rest if not a.startswith("--"))
    verses   = load_bible()
    rows     = parse_ref(query, verses)
    if not rows:
        print(vermelho(f"Nenhuma referência encontrada para '{query}'."), file=sys.stderr)
        sys.exit(1)
    num_part  = query.rpartition(" ")[2]
    ref_label = f"{rows[0]['book']} {num_part}" if num_part else rows[0]["book"]
    print(slides_read(rows, ref_label, no_cover, resp))

elif cmd == "terminal":
    query  = " ".join(rest)
    verses = load_bible()
    rows   = parse_ref(query, verses)
    if not rows:
        print(vermelho(f"Nenhuma referência encontrada para '{query}'."), file=sys.stderr)
        sys.exit(1)
    num_part  = query.rpartition(" ")[2]
    ref_label = f"{rows[0]['book']} {num_part}" if num_part else rows[0]["book"]
    terminal_read(rows, ref_label)

elif cmd == "search":
    word    = " ".join(rest)
    verses  = load_bible()
    results = [v for v in verses if word.lower() in v["text"].lower()]
    pattern = re.compile(re.escape(word), re.IGNORECASE)
    print(azul(f"{len(results)} ocorrência(s) de '{amarelo(word)}':"))
    print()
    for r in results:
        ref         = f"{r['book']} {r['chapter']}:{r['verse']}"
        highlighted = pattern.sub(lambda m: vermelho(m.group(0)), r["text"])
        print(f"{verde(f'{ref:25}')} {highlighted}")

PYEOF
}

# ==============================================================================
# Músicas
# ==============================================================================

_render_json_song() {
    local title="$1"
    MUSICAS_FILE="$MUSICAS_FILE" python3 - "$title" <<'PYEOF'
import sys, json, os, re, signal

signal.signal(signal.SIGPIPE, signal.SIG_DFL)

musicas_file = os.environ["MUSICAS_FILE"]
target_title = sys.argv[1]

with open(musicas_file, encoding='utf-8') as f:
    categorias = json.load(f)

stanzas = None
for cat_songs in categorias.values():
    if target_title in cat_songs:
        stanzas = cat_songs[target_title]
        break

if stanzas is not None:
    m = re.match(r'^(\d+(?:-[A-Za-z])?)\s*-\s*.+$', target_title)
    if m:
        print(f"HINO {m.group(1)}")
        print("\n\n")
    elif re.search(r'salmo', target_title, re.IGNORECASE):
        print(target_title)
        print("\n\n")

    for stanza in stanzas:
        print(stanza)
        print("\n")
else:
    print(f"Erro: Música '{target_title}' não encontrada.", file=sys.stderr)
    sys.exit(1)
PYEOF
}

pick_song() {
    _ensure_musicas || return 1

    while true; do
        local chosen
        chosen=$(python3 -c "
import json, signal
signal.signal(signal.SIGPIPE, signal.SIG_DFL)
data = json.load(open('$MUSICAS_FILE'))
titulos = sorted(t for cat_songs in data.values() for t in cat_songs)
print('CLEAR - Limpar Tela')
print('\n'.join(titulos))
" | fzf --reverse --prompt="Música (ESC para sair): ")

        [ -z "$chosen" ] && break

        if [ "$chosen" = "CLEAR - Limpar Tela" ]; then
            clear_sent
        else
            _render_json_song "$chosen" | format_sent
        fi
    done
}

# ==============================================================================
# Adicionar / Editar Músicas (JSON <-> MD temporário)
# ==============================================================================

_update_json_from_md() {
    local action="$1"
    local title="$2"
    local tmp_md="$3"

    MUSICAS_FILE="$MUSICAS_FILE" python3 - "$action" "$title" "$tmp_md" <<'PYEOF'
import json, sys, os, re

musicas_file = os.environ["MUSICAS_FILE"]
action = sys.argv[1]
title = sys.argv[2]
tmp_md = sys.argv[3]

CATEGORIAS = ["Hinos", "Salmos", "Cânticos Espirituais"]

if os.path.exists(musicas_file):
    with open(musicas_file, 'r', encoding='utf-8') as f:
        try:
            songs = json.load(f)
        except json.JSONDecodeError:
            songs = {}
else:
    songs = {}

for c in CATEGORIAS:
    songs.setdefault(c, {})

def find_category(title):
    for cat in CATEGORIAS:
        if title in songs.get(cat, {}):
            return cat
    return None

def infer_category(title):
    if re.match(r'^\d+(?:-[A-Za-z])?\s*-\s*.+$', title):
        return "Hinos"
    if re.search(r'salmo', title, re.IGNORECASE):
        return "Salmos"
    return "Cânticos Espirituais"

if action == "extract":
    cat = find_category(title)
    text = "\n\n".join(songs.get(cat, {}).get(title, [])) if cat else ""
    with open(tmp_md, 'w', encoding='utf-8') as f:
        f.write(f"# {title}\n\n{text}\n")

elif action == "save":
    with open(tmp_md, 'r', encoding='utf-8') as f:
        content = f.read()

    content = re.sub(r'^#.*\n?', '', content, flags=re.MULTILINE)
    stanzas = [b.strip() for b in re.split(r'\n{2,}', content) if b.strip()]

    if stanzas:
        cat = find_category(title) or infer_category(title)
        songs[cat][title] = stanzas
        with open(musicas_file, 'w', encoding='utf-8') as f:
            json.dump(songs, f, ensure_ascii=False, indent=2)
        print(f"✅ Música salva com sucesso em '{cat}'!")
    else:
        print("⚠️  Nenhuma estrofe encontrada. Alteração descartada.")
PYEOF
}

add_song() {
    _ensure_musicas || return 1
    echo -n "Digite o título da nova música: "
    read -r title
    [ -z "$title" ] && return 1

    local tmp_md="/tmp/culto_musica_$$.md"
    echo -e "# $title\n\nEstrofe 1...\n\nEstrofe 2..." > "$tmp_md"

    ${EDITOR} -c "set filetype=markdown" "$tmp_md"
    _update_json_from_md "save" "$title" "$tmp_md"
    rm -f "$tmp_md"
}

edit_song() {
    _ensure_musicas || return 1

    local chosen
    chosen=$(python3 -c "
import json, signal
signal.signal(signal.SIGPIPE, signal.SIG_DFL)
data = json.load(open('$MUSICAS_FILE'))
titulos = sorted(t for cat_songs in data.values() for t in cat_songs)
print('\n'.join(titulos))
" | fzf --reverse --prompt="Editar Música: ")

    [ -z "$chosen" ] && return 0

    local tmp_md="/tmp/culto_musica_$$.md"
    _update_json_from_md "extract" "$chosen" "$tmp_md"
    ${EDITOR} -c "set filetype=markdown" "$tmp_md"
    _update_json_from_md "save" "$chosen" "$tmp_md"
    rm -f "$tmp_md"
}

# ==============================================================================
# Bíblia — wrappers para sent
# ==============================================================================

get_bible() {
    local query="$1"
    local no_cover="$2"
    local resp="$3"

    local flags=""
    [ "$no_cover" = "true" ] && flags="$flags --no-cover"
    [ "$resp"     = "true" ] && flags="$flags --resp"

    _ensure_bible || exit 1
    _bible_python read $flags "$query" | format_sent
}

# ==============================================================================
# Liturgia
# ==============================================================================

show_menu() {
    while true; do
        local sel
        sel=$(printf '%s\n' "${LITURGIA_FIXA[@]}" \
            | fzf --reverse --prompt="Liturgia (ESC para sair): ")
        [ -z "$sel" ] && break

        local tp val
        tp="${sel%%:*}"
        val=$(printf '%s' "${sel#*:}" | sed 's/^[[:space:]]*//')

        case "$tp" in
            T)  printf '%s' "$val" | format_sent ;;
            AG) printf '%s' "$AGENDA_TEXTO" | format_sent ;;
            C)  clear_sent ;;
            *)  echo "Tipo desconhecido: $tp" >&2 ;;
        esac
    done
}

# ==============================================================================
# Lógica Principal
# ==============================================================================

case "${1:-}" in
    m|menu)           show_menu ;;

    cm)               pick_song ;;
    as|add-song)      add_song ;;
    es|edit-song)     edit_song ;;

    b|bible)          get_bible "$2" "false" "false" ;;
    bd|bible-clean)   get_bible "$2" "true"  "false" ;;
    r|resp)           get_bible "$2" "false" "true"  ;;

    v|ver)
        _ensure_bible || exit 1
        _bible_python terminal "$2" | less -R -i
        ;;
    s|search)
        _ensure_bible || exit 1
        _bible_python search "$2" | less -R -i
        ;;

    t|text)           printf '%s' "$2" | format_sent ;;
    ag|agenda)        printf '%s' "$AGENDA_TEXTO" | format_sent ;;
    x|clear)          clear_sent ;;

    i|index)
        awk 'BEGIN{RS="\n\n+"; c=1} {
            gsub(/^\s+|\s+$/, "")
            if (length($0) > 0) printf "[%02d] %.40s...\n", c++, $0
        }' "$SENT_FILE"
        ;;

    p|pix)
        if [ ! -f "$PIX_FILE" ]; then
            echo "Erro: Arquivo PIX não encontrado em $PIX_FILE" >&2
            exit 1
        fi
        nohup xdg-open "$PIX_FILE" >/dev/null 2>&1 &
        ;;

    *)
        cat <<'USAGE'
culto <comando> [arg]

  m / menu          Liturgia (fzf)

  cm                Selecionar música (fzf)
  as / add-song     Adicionar nova música (via Vim)
  es / edit-song    Editar música existente (via Vim)

  b / bible <ref>   Bíblia (com capa)
  bd        <ref>   Bíblia (sem capa)
  r / resp  <ref>   Responsivo (DIRIGENTE / IGREJA)
  v / ver   <ref>   Bíblia no terminal
  s / search <pal>  Buscar na bíblia

  t / text  <texto> Texto livre
  ag / agenda       Agenda da semana
  i / index         Índice dos slides atuais
  x / clear         Limpar tela do sent
  p / pix           Abrir QR code PIX
USAGE
        ;;
esac
