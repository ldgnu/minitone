# amusic-cli 🎵

TUI para Apple Music en Linux. Controla Cider desde la terminal, estilo spotify-tui.

![screenshot](https://i.imgur.com/placeholder.png)

## Requisitos

- **[Cider](https://cider.sh)** corriendo con RPC habilitado
  - Settings → Connectivity → Websocket API → **Enable**
  - (Opcional) Configurar API token en Settings → Connectivity → Manage External Application Access

## Instalación

### Arch Linux (AUR)

```bash
# Desde AUR (cuando esté publicado)
yay -S amusic-cli-bin

# O manual
git clone https://github.com/ldgnu/amusic-cli
cd amusic-cli
makepkg -si
```

### go install

```bash
go install github.com/ldgnu/amusic-cli/cmd/amusic@latest
```

### Manual

```bash
git clone https://github.com/ldgnu/amusic-cli
cd amusic-cli
go build -ldflags="-s -w" -o amusic ./cmd/amusic/
sudo install -m755 amusic /usr/local/bin/
```

## Uso

```bash
# Iniciar (Cider debe estar corriendo)
amusic

# Con token personalizado
CIDER_API_TOKEN=tu-token amusic

# Con storefront específico
CIDER_STOREFRONT=es amusic

# Tema específico
AMUSIC_THEME=catppuccin amusic
```

### Controles

| Tecla | Acción |
|-------|--------|
| `Space` | Play / Pause |
| `n` | Siguiente |
| `p` | Anterior |
| `+` / `-` | Subir / Bajar volumen |
| `s` | Buscar canciones |
| `l` | Biblioteca (playlists) |
| `w` | Ver cola |
| `y` | Ver letra |
| `t` | Cambiar tema |
| `z` | Alternar shuffle |
| `x` | Alternar repeat |
| `q` | Volver / Salir |
| `↑↓` / `jk` | Navegar listas |
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

- [ ] Atajo para like/dislike
- [ ] Artwork en terminal (sixel/kitty)
- [ ] Modo Chromium headless (sin Cider)
- [ ] Más temas
- [ ] Config file (`~/.config/amusic/config.json`)
