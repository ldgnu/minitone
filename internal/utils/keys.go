package utils

type Binding struct {
	Key  string
	Help string
}

type KeyMap struct {
	Search       Binding
	Enter        Binding
	Escape       Binding
	Up           Binding
	Down         Binding
	Tab          Binding
	ShiftTab     Binding
	PlayPause    Binding
	Stop         Binding
	VolumeUp     Binding
	VolumeDown   Binding
	SeekForward  Binding
	SeekBackward Binding
	QueuePanel   Binding
	HistoryPanel Binding
	Favorites    Binding
	RadioSource  Binding
	YouTubeSourc Binding
	NavidromeSrc Binding
	LocalSource  Binding
	Theme        Binding
	Help         Binding
	Quit         Binding
	Mute         Binding
	Shuffle      Binding
	Repeat       Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Search:       Binding{Key: "/", Help: "/ search"},
		Enter:        Binding{Key: "enter", Help: "enter play"},
		Escape:       Binding{Key: "esc", Help: "esc cancel"},
		Up:           Binding{Key: "k", Help: "↑ up"},
		Down:         Binding{Key: "j", Help: "↓ down"},
		Tab:          Binding{Key: "tab", Help: "tab next group"},
		ShiftTab:     Binding{Key: "shift+tab", Help: "stabc prev group"},
		PlayPause:    Binding{Key: "space", Help: "space play/pause"},
		Stop:         Binding{Key: "s", Help: "s stop"},
		VolumeUp:     Binding{Key: "+", Help: "+ vol up"},
		VolumeDown:   Binding{Key: "-", Help: "- vol down"},
		SeekForward:  Binding{Key: "right", Help: "→ seek +5s"},
		SeekBackward: Binding{Key: "left", Help: "← seek -5s"},
		QueuePanel:   Binding{Key: "ctrl+j", Help: "ctrl+j queue"},
		HistoryPanel: Binding{Key: "ctrl+h", Help: "ctrl+h history"},
		Favorites:    Binding{Key: "ctrl+f", Help: "ctrl+f favorites"},
		RadioSource:  Binding{Key: "ctrl+r", Help: "ctrl+r radio"},
		YouTubeSourc: Binding{Key: "ctrl+y", Help: "ctrl+y youtube"},
		NavidromeSrc: Binding{Key: "ctrl+n", Help: "ctrl+n navidrome"},
		LocalSource:  Binding{Key: "ctrl+l", Help: "ctrl+l library"},
		Theme:        Binding{Key: "t", Help: "t theme"},
		Help:         Binding{Key: "?", Help: "? help"},
		Quit:         Binding{Key: "q", Help: "q quit"},
		Mute:         Binding{Key: "m", Help: "m mute"},
		Shuffle:      Binding{Key: "S", Help: "S shuffle"},
		Repeat:       Binding{Key: "R", Help: "R repeat"},
	}
}
