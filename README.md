# Culto

TUI em Go para organizar a liturgia, consultar a Bíblia e projetar slides usando `sent`. A aplicação mantém compatibilidade com `biblia.json` e `musicas.json` usados pelo script Bash anterior, arquivado em `legacy/culto.sh`. O comando `culto` do Bash executa esta TUI por meio da função configurada em `~/.bashrc`.

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
- **Ctrl+L** limpa a projeção do `sent` a partir de qualquer tela da TUI, sem sair dela.
- Na visualização, **p** projeta e **g** permite digitar o número do slide e saltar diretamente no `sent`, sem sair da tela. **←/→** passam ao slide anterior/seguinte. **Esc** cancela a entrada.
- A visualização indica o número de cada slide; use esse número depois de apertar **g**.
- Em músicas, **e** abre o Vim/Neovim, **t** renomeia o título e **d** remove a música após confirmação digitando `REMOVER`.
- Na leitura bíblica, **v** permite ajustar a referência. Depois de informar um livro e capítulo, basta digitar o número do versículo ou intervalo (por exemplo, `10` ou `10-12`) para continuar no mesmo capítulo.
- No editor, use `:wq` para salvar e sair; `:q!` para descartar.
- **Ctrl+C** encerra a interface.

Na visualização de uma música, cada parte é identificada com o número do slide e o número da estrofe. Se houver uma capa automática (hinos e salmos), ela aparece como slide 01 e a primeira estrofe como slide 02. Esses rótulos são apenas informativos e não aparecem na projeção.

O menu inicial fica centralizado e evita repetir ações já acessíveis por atalhos globais. Erros aparecem na prévia, mas não podem ser enviados à projeção.

A TUI não encerra processos `sent`. Ao abrir uma nova apresentação em uma sessão Hyprland, o app inicia o `sent` diretamente no monitor `HDMI-A-1`; se esse monitor não estiver conectado, mostra um erro em vez de abrir a apresentação em outra tela. Se não puder atualizar uma apresentação existente, também mostra um erro em vez de matar processos ou abrir outra instância por engano. A gravação de letras é atômica e arquivos JSON inválidos não são substituídos. Renomeações e remoções de músicas preservam o restante do arquivo e são gravadas atomicamente.

## Testes

```sh
go test ./...
```
