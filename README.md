# minitone ♪

TUI for Apple Music via Cider, like spotify-tui.

by [ldgnu](https://github.com/ldgnu)

## Requisitos

- **[Cider](https://cider.sh)** bajado y corriendo con RPC activado
  - Settings → Connectivity → Websocket API → **Enable**
  - (Opcional) Token de API en Manage External Application Access

## Instalación

### Arch Linux (AUR — pronto)

```bash
yay -S minitone-bin

# O manual
git clone https://github.com/ldgnu/minitone
cd minitone
makepkg -si
```

### go install

```bash
go install github.com/ldgnu/minitone/cmd/minitone@latest
```

### Manual

```bash
git clone https://github.com/ldgnu/minitone
cd minitone
go build -ldflags="-s -w" -o minitone ./cmd/minitone/
sudo cp minitone /usr/local/bin/
```

## Uso

```bash
# Arrancá (Cider tiene que estar abierto)
minitone

# Con token
CIDER_API_TOKEN=token minitone

# Con storefront argento
CIDER_STOREFRONT=ar minitone

# Temita lindo
AMUSIC_THEME=catppuccin minitone
```

### Controles

| Tecla | Acción |
|-------|--------|
| `Space` | Play / Pausa |
| `n` | Siguiente tema |
| `p` | Tema anterior |
| `+` / `-` | Volumen pa' arriba / abajo |
| `s` | Buscar canciones |
| `l` | Biblioteca (playlists) |
| `w` | Ver cola |
| `y` | Ver letra |
| `t` | Cambiar tema visual |
| `z` | Mezclar (shuffle) |
| `x` | Repetir |
| `q` | Volver / Salir |
| `↑↓` / `jk` | Moverse en las listas |
| `Enter` | Buscar |
| `Tab` | Cambiar categoría |

### Temas

`system`, `tokyonight`, `everforest`, `ayu`, `catppuccin`,
`catppuccin-macchiato`, `gruvbox`, `kanagawa`, `nord`, `matrix`, `one-dark`

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `CIDER_API_BASE` | `http://localhost:10767` | URL del RPC de Cider |
| `CIDER_API_TOKEN` | — | Token de API de Cider |
| `CIDER_STOREFRONT` | `us` | Storefront de Apple Music |
| `AMUSIC_THEME` | `system` | Tema de colores |

## TODO

- [ ] Like / dislike
- [ ] Carátula en la terminal (sixel / kitty)
- [ ] Modo Chromium headless (sin depender de Cider)
- [ ] Más temas
- [ ] Config file (`~/.config/minitone/config.json`)
