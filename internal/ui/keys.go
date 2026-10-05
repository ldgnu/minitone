package ui

import "strings"

// backspaceKey is the key name Bubble Tea reports for backspace.
const backspaceKey = "backspace"

// action is one logical command. Keeping actions named (instead of scattering
// raw key strings) is what makes the contextual help and the keybindings
// config honest: the footer and `?` are generated from this table.
type action string

const (
	actNone          action = ""
	actPlay          action = "play"
	actPause         action = "pause"
	actStop          action = "stop"
	actNext          action = "next"
	actPrev          action = "prev"
	actEnqueue       action = "enqueue"
	actEnqueueAll    action = "enqueue-all"
	actFavorite      action = "favorite"
	actDetails       action = "details"
	actDelete        action = "delete"
	actClearQueue    action = "clear-queue"
	actMoveUp        action = "move-up"
	actMoveDown      action = "move-down"
	actVolumeUp      action = "volume-up"
	actVolumeDown    action = "volume-down"
	actMute          action = "mute"
	actSeekFwd       action = "seek-forward"
	actSeekBack      action = "seek-back"
	actCycleTheme    action = "theme"
	actCycleRepeat   action = "repeat"
	actToggleShuffle action = "shuffle"
	actToggleVideo   action = "video"
	actQueue         action = "queue"
	actFavorites     action = "favorites"
	actHistory       action = "history"
	actLibrary       action = "library"
	actHelp          action = "help"
	actHelpKey       action = "help-key"
	actRescan        action = "rescan"
	actQuit          action = "quit"
)

// binding is a key plus a short label for the help/footer.
type binding struct {
	key  string
	help string
}

// KeyMap resolves key names to actions.
//
// Browse-mode single-key actions are listed separately from the global ctrl
// bindings so typing a query is never hijacked, and so the same action can be
// reached both ways (e.g. ctrl+j / Tab).
type KeyMap struct {
	// global bindings work in every context
	Enter     binding
	Esc       binding
	Tab       binding
	Space     binding
	Left      binding
	Right     binding
	Up        binding
	Down      binding
	Plus      binding
	Minus     binding
	Queue     binding
	Favorites binding
	History   binding
	Library   binding
	Help      binding
	HelpKey   binding
	Quit      binding
	QuitKey   binding
	Theme     binding
	Video     binding
	Repeat    binding
	Shuffle   binding
	Rescan    binding

	// browse-mode single-key actions
	Enqueue    binding
	EnqueueAll binding
	Favorite   binding
	Details    binding
	Delete     binding
	ClearQueue binding
	MoveUp     binding
	MoveDown   binding
	NextKey    binding
	PrevKey    binding
	Mute       binding
	Stop       binding
}

// NewKeyMap returns the default bindings.
func NewKeyMap() KeyMap {
	return KeyMap{
		Enter:     binding{"enter", "play selected"},
		Esc:       binding{"esc", "back"},
		Tab:       binding{"tab", "next source / panel"},
		Space:     binding{" ", "play / pause"},
		Left:      binding{"left", "seek -5s"},
		Right:     binding{"right", "seek +5s"},
		Up:        binding{"up", "up"},
		Down:      binding{"down", "down"},
		Plus:      binding{"+", "volume +"},
		Minus:     binding{"-", "volume -"},
		Queue:     binding{"ctrl+j", "queue"},
		Favorites: binding{"ctrl+f", "favorites"},
		History:   binding{"ctrl+h", "history"},
		Library:   binding{"ctrl+l", "library"},
		Help:      binding{"ctrl+/", "help"},
		// "?" is the portable one: ctrl+/ is not reported by every terminal.
		HelpKey: binding{"?", "help"},
		Quit:    binding{"ctrl+c", "quit"},
		QuitKey: binding{"q", "quit"},
		Theme:   binding{"ctrl+t", "theme"},
		Video:   binding{"ctrl+v", "video"},
		Repeat:  binding{"ctrl+r", "repeat"},
		Shuffle: binding{"ctrl+u", "shuffle"},
		Rescan:  binding{"ctrl+s", "rescan library"},

		Enqueue:    binding{"a", "add to queue"},
		EnqueueAll: binding{"A", "add all results"},
		Favorite:   binding{"f", "favorite"},
		Details:    binding{"i", "details"},
		Delete:     binding{"d", "remove"},
		ClearQueue: binding{"c", "clear queue"},
		MoveUp:     binding{"J", "move up"},
		MoveDown:   binding{"K", "move down"},
		NextKey:    binding{"n", "next"},
		PrevKey:    binding{"p", "previous"},
		Mute:       binding{"m", "mute"},
		Stop:       binding{"s", "stop"},
	}
}

// Apply overrides bindings from config ({"queue": "ctrl+q"}).
func (k *KeyMap) Apply(overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}
	set := func(b *binding, name string) {
		if v, ok := overrides[name]; ok && strings.TrimSpace(v) != "" {
			b.key = strings.ToLower(strings.TrimSpace(v))
		}
	}
	set(&k.Enter, "enter")
	set(&k.Esc, "esc")
	set(&k.Tab, "tab")
	set(&k.Space, "play_pause")
	set(&k.Left, "seek_back")
	set(&k.Right, "seek_forward")
	set(&k.Plus, "volume_up")
	set(&k.Minus, "volume_down")
	set(&k.Queue, "queue")
	set(&k.Favorites, "favorites")
	set(&k.History, "history")
	set(&k.Library, "library")
	set(&k.Help, "help")
	set(&k.HelpKey, "help")
	set(&k.Theme, "theme")
	set(&k.Video, "video")
	set(&k.Repeat, "repeat")
	set(&k.Shuffle, "shuffle")
	set(&k.Enqueue, "enqueue")
	set(&k.EnqueueAll, "enqueue_all")
	set(&k.Favorite, "favorite")
	set(&k.Details, "details")
	set(&k.Delete, "delete")
	set(&k.ClearQueue, "clear_queue")
	set(&k.MoveUp, "move_up")
	set(&k.MoveDown, "move_down")
	set(&k.NextKey, "next")
	set(&k.PrevKey, "prev")
	set(&k.Mute, "mute")
	set(&k.Stop, "stop")
	set(&k.Rescan, "rescan")
}

// resolve maps a key name to an action, using the configured bindings.
func (k KeyMap) resolve(key string) action {
	pairs := []struct {
		b binding
		a action
	}{
		{k.Enter, actPlay},
		{k.Esc, actNone}, // handled separately: it means "cancel"
		{k.Tab, actNone}, // context dependent
		{k.Space, actPause},
		{k.Left, actSeekBack},
		{k.Right, actSeekFwd},
		{k.Plus, actVolumeUp},
		{k.Minus, actVolumeDown},
		{k.Queue, actQueue},
		{k.Favorites, actFavorites},
		{k.History, actHistory},
		{k.Library, actLibrary},
		{k.Help, actHelp},
		{k.HelpKey, actHelpKey},
		{k.Quit, actQuit},
		{k.Theme, actCycleTheme},
		{k.Video, actToggleVideo},
		{k.Repeat, actCycleRepeat},
		{k.Shuffle, actToggleShuffle},
		{k.Rescan, actRescan},
	}
	for _, p := range pairs {
		if p.b.key == key {
			return p.a
		}
	}
	return actNone
}

// browseActions are the single-key actions available in list focus.
func (k KeyMap) browseActions() map[string]action {
	return map[string]action{
		k.Enqueue.key:    actEnqueue,
		k.EnqueueAll.key: actEnqueueAll,
		k.Favorite.key:   actFavorite,
		k.Details.key:    actDetails,
		k.Delete.key:     actDelete,
		k.ClearQueue.key: actClearQueue,
		k.MoveUp.key:     actMoveUp,
		k.MoveDown.key:   actMoveDown,
		k.NextKey.key:    actNext,
		k.PrevKey.key:    actPrev,
		k.Mute.key:       actMute,
		k.Stop.key:       actStop,
		"j":              actNone, // navigation, handled separately
		"k":              actNone,
	}
}

// hint is one "[key] label" pair for the contextual footer.
type hint struct {
	key   string
	label string
}

// hintsFor returns the most relevant hints for the current context.
//
// Deliberately short: the full list lives behind `?`.
func (m Model) hintsFor() []hint {
	k := m.keys
	if m.panel != PanelNone {
		return m.panelHints()
	}

	if m.notice.active() {
		return m.noticeHints()
	}

	switch m.focus {
	case FocusSearch:
		var h []hint
		if m.search.total() > 0 {
			h = append(h, hint{"tab", "browse"}, hint{k.Enter.key, "play"})
		} else {
			h = append(h, hint{"/term", "search"}, hint{"/yt term", "youtube"})
		}
		if m.queue.Len() > 0 {
			h = append(h, hint{k.Queue.key, "queue"})
		}
		return append(h, hint{k.Help.key, "help"}, hint{k.QuitKey.key, "quit"})
	default:
		h := []hint{
			{k.Enqueue.key, "queue"},
			{k.Favorite.key, "fav"},
			{k.NextKey.key, "next"},
			{k.Space.key, "pause"},
			{k.Esc.key, "search"},
			{k.Help.key, "help"},
		}
		if m.queue.Len() > 0 {
			h = append(h, hint{k.Queue.key, "queue"})
		}
		return h
	}
}

func (m Model) panelHints() []hint {
	k := m.keys
	switch m.panel {
	case PanelQueue:
		return []hint{
			{k.Enter.key, "play"},
			{k.Delete.key, "remove"},
			{hintKey(k.MoveUp), "move"},
			{k.EnqueueAll.key, "enqueue all"},
			{k.ClearQueue.key, "clear"},
			{k.Esc.key, "close"},
		}
	case PanelFavorites, PanelHistory:
		return []hint{
			{k.Enter.key, "play"},
			{k.Enqueue.key, "enqueue"},
			{k.Favorite.key, "favorite"},
			{k.Delete.key, "remove"},
			{k.Esc.key, "close"},
		}
	case PanelLibrary:
		return []hint{
			{k.Enter.key, "open"},
			{k.EnqueueAll.key, "enqueue all"},
			{k.Rescan.key, "rescan"},
			{k.Esc.key, "close"},
		}
	case PanelDetails:
		return []hint{{k.Esc.key, "close"}}
	default:
		return []hint{{k.Help.key, "close"}}
	}
}

func (m Model) noticeHints() []hint {
	if m.notice.retryable {
		return []hint{{"r", "retry"}, {"esc", "dismiss"}}
	}
	if m.notice.action != "" {
		return []hint{{"esc", "dismiss"}}
	}
	return nil
}

func hintKey(b binding) string {
	if b.key == "J" || b.key == "K" {
		return b.key
	}
	return b.key
}

// helpRow is one line of the help overlay.
type helpRow struct {
	key  string
	desc string
}

// helpLabel maps a binding key to a printable label for the help overlay.
// Arrow keys and "space" have no printable equivalent, so they are spelled out.
func helpLabel(b binding) string {
	switch b.key {
	case "left":
		return "←"
	case "right":
		return "→"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case " ":
		return "space"
	case "esc":
		return "esc"
	default:
		return b.key
	}
}

func (m Model) helpRows() []helpRow {
	k := m.keys
	rows := []helpRow{
		{"── search ──", ""},
		{"type", "search (all sources)"},
		{"/youtube term", "search only YouTube"},
		{"/radio term", "search only Radio Browser"},
		{"/local term", "search only the local library"},
		{"esc", "clear search / back to search"},
		{k.Tab.key, "next source group · next panel"},
		{"↑ ↓ j k", "move in results"},

		{"── playback ──", ""},
		{k.Enter.key, "play selected"},
		{helpLabel(k.Space), "play / pause"},
		{k.NextKey.key + " / " + k.PrevKey.key, "next / previous track"},
		{helpLabel(k.Left) + " / " + helpLabel(k.Right), "seek -5s / +5s"},
		{k.Plus.key + " / " + k.Minus.key, "volume"},
		{k.Mute.key, "mute"},
		{k.Stop.key, "stop"},

		{"── queue ──", ""},
		{k.Enqueue.key, "add selected to queue"},
		{k.EnqueueAll.key, "add all results"},
		{k.Queue.key, "open queue"},
		{k.MoveUp.key + " / " + k.MoveDown.key, "move queue item"},
		{k.Delete.key, "remove from queue"},
		{k.ClearQueue.key, "clear queue"},
		{k.Shuffle.key, "shuffle"},
		{k.Repeat.key, "repeat off → all → one"},

		{"── library ──", ""},
		{k.Library.key, "browse local library"},
		{k.Rescan.key, "rescan library"},
		{k.Favorite.key, "add/remove favorite"},
		{k.Details.key, "track details"},
		{k.Favorites.key, "favorites"},
		{k.History.key, "history"},

		{"── app ──", ""},
		{k.Theme.key, "cycle theme"},
		{k.Video.key, "video mode"},
		{"? / " + k.Help.key, "this help"},
		{k.QuitKey.key, "quit (browse mode)"},
		{k.Quit.key, "quit (always works)"},
	}
	return rows
}
