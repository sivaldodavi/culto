# Culto

TUI em Go para organizar a liturgia, consultar a Bíblia e projetar slides usando `sent`. A aplicação mantém compatibilidade com `biblia.json` e `musicas.json` usados pelo script Bash `~/.local/bin/culto`; esse script não é alterado.

## Requisitos

- Go 1.23 ou posterior
- `sent` para projetar
- `xdotool` para atualizar uma janela `sent` já aberta
- `pgrep` para evitar iniciar uma segunda instância quando `sent` já estiver aberto
- `xdg-open` para abrir a imagem PIX

## Executar

```sh
cd ~/.local/dev/culto
go run ./cmd/culto
```

Para gerar um executável separado do script atual:

```sh
go build -o ~/.local/bin/culto-tui ./cmd/culto
```

## Dados

Por padrão, são usados os arquivos existentes em `~/Dufs/Igreja/Projeção`:

```text
biblia.json
musicas.json
pix-igreja.png
```

É possível configurar outro diretório/arquivo por variáveis de ambiente:

```sh
CULTO_DIR="$HOME/caminho/dos/dados" go run ./cmd/culto
```

Sobrescritas específicas: `CULTO_BIBLE_FILE`, `CULTO_MUSIC_FILE`, `CULTO_PIX_FILE` e `CULTO_SENT_FILE`.

## Teclas

- **↑/↓** navega; **/** filtra; **Enter** seleciona.
- **Esc** volta; **q** volta ou sai do menu inicial.
- Na visualização, **p** projeta; em músicas, **e** edita.
- No editor, **Ctrl+S** salva e **Esc** descarta.
- **Ctrl+C** encerra a interface.

Na visualização de uma música, cada parte é identificada com o número do slide e o número da estrofe. Se houver uma capa automática (hinos e salmos), ela aparece como slide 01 e a primeira estrofe como slide 02. Esses rótulos são apenas informativos e não aparecem na projeção.

A TUI não encerra processos `sent`. Se não puder atualizar a apresentação existente, exibe um erro em vez de matar processos ou abrir outra instância por engano. A gravação de letras é atômica e arquivos JSON inválidos não são substituídos.

## Testes

```sh
go test ./...
```
