package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey is the single entry point for keyboard input.
//
// There is exactly one dispatcher (the old code had two, and the second one
// was unreachable). Context decides the meaning:
//
//	panel open   → panel keys
//	esc          → always "cancel / go back"
//	global keys  → playback + panel switches (ctrl combos, enter, space, arrows)
//	focus=search → every printable key is typed
//	focus=list   → single-key actions (f a d i c J K n p …); any other printable
//	               key falls back to typing, so a search term can still contain
//	               "s", "d", "n"…
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	k := m.keys

	// ctrl+c always quits, whatever the context.
	if key == k.Quit.key {
		m.saveSession()
		return m, tea.Quit
	}

	if m.panel != PanelNone {
		return m.handlePanelKey(key)
	}

	handled, cmd, quit := m.handleGlobalKey(key)
	if quit {
		m.saveSession()
		return m, tea.Quit
	}
	if handled {
		return m, cmd
	}

	if key == k.Esc.key {
		return m, m.handleEscape()
	}

	// A retryable error takes precedence: r retries the failed search.
	if m.notice.active() && m.notice.retryable && key == "r" {
		return m, m.retrySearch()
	}

	// "d" on the restored-session notice clears the stored session, so a stale
	// queue can be discarded with the same key that works everywhere else.
	if m.notice.active() && m.notice.retryable && key == m.keys.Delete.key {
		m.clearSession()
		return m, nil
	}

	if m.focus == FocusList {
		return m.handleListKey(key)
	}
	return m, m.handleSearchKey(key)
}

// handleGlobalKey processes bindings that work everywhere. It reports whether
// the key was consumed, plus any command to run and whether to quit.
func (m *Model) handleGlobalKey(key string) (handled bool, cmd tea.Cmd, quit bool) {
	k := m.keys

	// "?" is checked before anything else: it is the portable help key and
	// must never end up in the query.
	if key == k.HelpKey.key && m.panel != PanelHelp {
		m.focusBeforePanel = m.focus
		m.panel = PanelHelp
		return true, nil, false
	}

	switch k.resolve(key) {
	case actPause:
		// Space is both "play/pause" and a character. While the search box is
		// focused it belongs to the query — otherwise "lofi girl" is
		// impossible to type.
		if m.focus == FocusSearch {
			return false, nil, false
		}
		return true, m.togglePause(), false
	case actSeekFwd:
		if m.player != nil {
			m.player.Seek(5)
		}
		return true, nil, false
	case actSeekBack:
		if m.player != nil {
			m.player.Seek(-5)
		}
		return true, nil, false
	case actVolumeUp:
		// "+" and "-" are characters too: they only control the volume when
		// they cannot be part of a search term.
		if m.focus == FocusSearch && m.search.query != "" {
			return false, nil, false
		}
		m.changeVolume(5)
		return true, nil, false
	case actVolumeDown:
		if m.focus == FocusSearch && m.search.query != "" {
			return false, nil, false
		}
		m.changeVolume(-5)
		return true, nil, false
	case actQueue:
		return true, m.openPanelCmd(PanelQueue), false
	case actFavorites:
		return true, m.openPanelCmd(PanelFavorites), false
	case actHistory:
		return true, m.openPanelCmd(PanelHistory), false
	case actLibrary:
		return true, m.openPanelCmd(PanelLibrary), false
	case actPlaylists:
		return true, m.openPanelCmd(PanelPlaylists), false
	case actHelp:
		if m.panel != PanelHelp {
			m.focusBeforePanel = m.focus
		}
		m.panel = PanelHelp
		return true, nil, false
	case actHelpKey:
		// "?" must not be swallowed by the search box: it is the help key that
		// works in every terminal (ctrl+/ is not reported by all of them).
		if m.panel != PanelHelp {
			m.focusBeforePanel = m.focus
		}
		m.panel = PanelHelp
		return true, nil, false
	case actCycleTheme:
		m.toggleTheme()
		return true, nil, false
	case actToggleVideo:
		m.toggleVideo()
		return true, nil, false
	case actCycleRepeat:
		m.cycleRepeat()
		return true, nil, false
	case actToggleShuffle:
		m.toggleShuffle()
		return true, nil, false
	case actRescan:
		m.startScanCmd()
		return true, nil, false
	}

	switch key {
	case k.Enter.key:
		return true, m.enterCmd(), false
	case k.Tab.key:
		m.tab()
		return true, nil, false
	case "shift+tab":
		m.shiftTab()
		return true, nil, false
	case "up":
		m.focus = FocusList
		m.moveCursor(-1)
		mv, c := m.maybeFetchThumb()
		*m = mv
		return true, c, false
	case "down":
		m.focus = FocusList
		m.moveCursor(1)
		mv, c := m.maybeFetchThumb()
		*m = mv
		return true, c, false
	case k.QuitKey.key:
		// "q" quits from the list and from an empty search box. Once a query
		// exists it becomes a letter again, so searching for anything starting
		// with "q" still works.
		if m.focus == FocusList || m.search.query == "" {
			return true, nil, true
		}
	}
	return false, nil, false
}

// togglePause pauses/resumes, or starts the queue when nothing is loaded.
func (m *Model) togglePause() tea.Cmd {
	if m.player == nil {
		return nil
	}
	if m.player.Status().Song.Empty() {
		if m.queue.Len() == 0 {
			m.notice = infoNotice("nothing to play", "search something and press enter", "")
			return nil
		}
		m.playToken++
		if item := m.queue.Current(); item != nil {
			return m.resolveAndPlay(item.Song)
		}
		return nil
	}
	if err := m.player.TogglePause(); err != nil {
		m.notice = errorNotice("mpv is not responding", errText(err), "[esc] dismiss")
	}
	return nil
}

// enterCmd plays the highlighted result, or drills into the library.
func (m *Model) enterCmd() tea.Cmd {
	if m.panel == PanelLibrary {
		m.enterLibrary()
		return nil
	}
	if cmd := m.playSelected(); cmd != nil {
		return cmd
	}
	if m.queue.Len() > 0 {
		m.focus = FocusList
		m.notice = infoNotice("nothing selected", "press n to play the queue", "")
	}
	return nil
}

// openPanelCmd opens (or closes) an overlay panel and returns a command.
func (m *Model) openPanelCmd(p Panel) tea.Cmd {
	if m.panel == p {
		m.panel = PanelNone
		return nil
	}
	switch p {
	case PanelFavorites:
		if m.favs.Len() == 0 {
			m.notice = infoNotice("no favorites yet", "press f on a track to add one", "")
			return nil
		}
	case PanelPlaylists:
		if m.playls == nil || m.playls.Len() == 0 {
			m.notice = infoNotice("no playlists yet", "they are created from taste_genres/artists on start", "")
			return nil
		}
		m.plLevel = 0
		m.plIndex = 0
	case PanelHistory:
		if m.hist.Len() == 0 {
			m.notice = infoNotice("history is empty", "play something first", "")
			return nil
		}
	case PanelQueue:
		if m.queue.Len() == 0 {
			m.notice = infoNotice("queue is empty", "press a on a result", "")
			return nil
		}
	case PanelLibrary:
		m.startScanCmd()
	}

	m.focusBeforePanel = m.focus
	m.panel = p
	m.panelCursor = 0
	if p == PanelLibrary {
		m.lib.open = true
		m.lib.reset()
		m.refreshLibrarySections()
	}
	return nil
}

// startScanCmd launches a library rescan without blocking the UI.
func (m *Model) startScanCmd() {
	if m.libraryScanner == nil {
		return
	}
	m.notice = infoNotice("scanning library…", "the app stays usable while it runs", "")
	_ = m.startScan()
}

func (m *Model) refreshLibrarySections() {
	if m.libraryScanner != nil {
		m.lib.sections = m.libraryScanner.Groups()
		m.lib.status = scanStatusText(m.libraryScanner.Status())
	}
}

// handleEscape implements the single "back" behaviour, most specific first.
//
// Each case returns immediately so dismissing an error does not also move the
// focus: one esc, one effect.
func (m *Model) handleEscape() tea.Cmd {
	switch {
	case m.panel == PanelLibrary && m.lib.level != libSections:
		m.lib.reset()
		return nil
	case m.panel != PanelNone:
		m.lib.open = false
		m.panel = PanelNone
		m.focus = m.focusBeforePanel
		return nil
	case m.notice.active() && m.notice.kind == noticeError:
		m.notice = notice{}
		return nil
	case m.focus == FocusList:
		m.focus = FocusSearch
		return nil
	case m.search.term != "" || m.search.query != "":
		m.clearSearch()
		return nil
	}
	m.focus = FocusSearch
	return nil
}

func (m *Model) clearSearch() {
	m.search.query = ""
	m.search.reset()
	if m.sm != nil {
		m.sm.Cancel()
	}
	m.err = nil
	m.notice = notice{}
}

// handleSearchKey: printable keys extend the query, backspace shortens it.
func (m *Model) handleSearchKey(key string) tea.Cmd {
	if key == backspaceKey {
		m.backspace()
		return nil
	}
	if !isPrintable(key) {
		return nil
	}
	m.focus = FocusSearch
	m.search.query += key
	m.triggerSearch()
	return nil
}

// backspace removes the last character of the query and re-searches.
func (m *Model) backspace() {
	if m.search.query == "" {
		return
	}
	r := []rune(m.search.query)
	m.search.query = string(r[:len(r)-1])
	m.triggerSearch()
}

// handleListKey: navigation plus the single-key actions.
func (m Model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j":
		m.moveCursor(1)
		return m.maybeFetchThumb()
	case "k":
		m.moveCursor(-1)
		return m.maybeFetchThumb()
	}

	// Escape is the documented way back to typing from the list; any
	// printable key that is not a shortcut also falls through to the box.
	if act, ok := m.keys.browseActions()[key]; ok {
		switch act {
		case actEnqueue:
			m.enqueueSelected()
			return m, nil
		case actEnqueueAll:
			m.enqueueAll()
			return m, nil
		case actFavorite:
			m.toggleFavorite()
			return m, nil
		case actDetails:
			m.showDetails()
			return m, nil
		case actDelete:
			m.removePlayingFromQueue()
			return m, nil
		case actClearQueue:
			m.clearQueue()
			return m, nil
		case actMoveUp:
			m.moveQueueItem(-1)
			return m, nil
		case actMoveDown:
			m.moveQueueItem(1)
			return m, nil
		case actNext:
			return m, m.playNext()
		case actPrev:
			return m, m.playPrev()
		case actMute:
			m.toggleMute()
			return m, nil
		case actStop:
			m.stop()
			return m, nil
		}
		return m, nil
	}

	// Anything else printable means "I am typing": fall back to the search box.
	if isPrintable(key) {
		m.focus = FocusSearch
		m.search.query += key
		m.triggerSearch()
		return m, nil
	}
	return m, nil
}

// stop stops playback without advancing the queue.
func (m *Model) stop() {
	if m.player == nil {
		return
	}
	_ = m.player.Stop()
	m.playToken++ // discard any in-flight resolve
	m.notice = okNotice("stopped", "", "")
}

// moveQueueItem moves the playing track up/down in the queue.
func (m *Model) moveQueueItem(delta int) {
	cur := m.queue.Cursor()
	if cur < 0 {
		m.notice = infoNotice("queue is empty", "add songs with a", "")
		return
	}
	target := cur + delta
	if target < 0 || target >= m.queue.Len() {
		return
	}
	if m.queue.Move(cur, target) {
		m.notice = okNotice("moved in queue", "", "")
	}
}

// tab has one job at a time, in this order:
//
//  1. if the search box has focus and there is something to browse, enter the
//     results list;
//  2. otherwise cycle the source group;
//  3. with a single (or no) group, cycle the source filter.
//
// Step 1 matters: tab must always be a predictable "move into the results".
func (m *Model) tab() {
	groups := m.search.groups()
	if m.focus == FocusSearch && len(groups) > 0 {
		m.focus = FocusList
		if m.search.group >= len(groups) {
			m.search.group = 0
		}
		return
	}
	if len(groups) > 1 {
		m.search.group = (m.search.group + 1) % len(groups)
		m.search.cursor = 0
		m.focus = FocusList
		return
	}
	m.cycleSource(1)
}

func (m *Model) shiftTab() {
	groups := m.search.groups()
	if m.focus == FocusSearch && len(groups) > 0 {
		m.focus = FocusList
		return
	}
	if len(groups) > 1 {
		m.search.group = (m.search.group - 1 + len(groups)) % len(groups)
		m.search.cursor = 0
		m.focus = FocusList
		return
	}
	m.cycleSource(-1)
}
