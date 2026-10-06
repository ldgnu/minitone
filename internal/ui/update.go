package ui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
	"github.com/ldgnu/minitone/internal/source/library"
	"github.com/ldgnu/minitone/internal/store"
)

// sessionSaveInterval is how often the session is written while playing.
const sessionSaveInterval = 30 * time.Second

// playToken identifies one "play this song" request.
//
// Resolving a YouTube stream can take seconds. Without a token, a resolve that
// started before a newer search/selection would still start playing the *old*
// song when it finally completed.
type playToken uint64

type playOKMsg struct {
	song  models.Song
	token playToken
}

type playErrMsg struct {
	err   error
	token playToken
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampCursors()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case searchEventMsg:
		cmd := m.applySearchEvent(msg.ev)
		return m, tea.Batch(cmd, m.waitSearchEvents())

	case songEndedMsg:
		cmd := m.autoAdvance()
		return m, tea.Batch(cmd, m.waitEnded())

	case tickMsg:
		return m.onTick()

	case thumbMsg:
		if msg.err == nil && msg.img != nil {
			if m.thumbs == nil {
				m.thumbs = map[string]image.Image{}
			}
			m.thumbs[msg.url] = msg.img
		}
		return m, nil

	case playOKMsg:
		return m.onPlayOK(msg)

	case playErrMsg:
		if msg.token != m.playToken {
			return m, nil // superseded by a newer selection
		}
		m.err = msg.err
		m.notice = errorNotice("cannot play "+m.lastSong.DisplayTitle(), errText(msg.err), "[esc] dismiss")
		return m, nil

	case playerErrMsg:
		if msg.err != nil {
			m.err = msg.err
			m.notice = errorNotice("playback error", errText(msg.err), "[n] next  [esc] dismiss")
		}
		return m, nil

	case scanMsg:
		m.onScan(msg.status)
		return m, m.waitScan()

	case sessionSaveMsg:
		m.saveSession()
		return m, nil

	case playlistRefreshMsg:
		m.onPlaylistRefresh(msg)
		return m, nil

	case error:
		m.err = msg
		m.notice = errorNotice("unexpected error", errText(msg), "[esc] dismiss")
		return m, nil
	}
	return m, nil
}

// ── ticks ───────────────────────────────────────────────────────────────────

func (m Model) onTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{tickCmd()}
	m.spinnerFrame++

	if m.notice.expired(time.Now()) {
		m.notice = notice{}
	}
	if m.restoreSessionOn && m.player != nil {
		if time.Since(m.lastSessionAt) > sessionSaveInterval {
			m.lastSessionAt = time.Now()
			cmds = append(cmds, func() tea.Msg { return sessionSaveMsg{} })
		}
	}
	if m.search.running && m.search.pendingSources() == nil {
		m.search.running = false
	}
	return m, tea.Batch(cmds...)
}

// ── search events ───────────────────────────────────────────────────────────

// applySearchEvent merges one streaming source update into the view state.
func (m *Model) applySearchEvent(ev search.Event) tea.Cmd {
	// A "cleared" event (empty query) just resets the view.
	if !ev.Started && ev.Source == "" {
		m.search.reset()
		m.notice = notice{}
		return nil
	}

	// Drop results for a query the user has already moved past.
	if ev.Query != m.search.term {
		return nil
	}

	if ev.Started {
		m.search.reset()
		m.search.term = ev.Query
		m.search.running = true
		m.search.startedAt = time.Now()
		for _, s := range m.activeSourceNames() {
			m.search.pending[s] = true
		}
		return nil
	}

	delete(m.search.pending, ev.Source)
	if ev.Err != nil {
		m.search.failures[ev.Source] = ev.Err
	} else if len(ev.Songs) > 0 {
		m.search.bySource[ev.Source] = ev.Songs
		delete(m.search.failures, ev.Source)
	}

	// Keep the cursor inside the new result set.
	m.clampCursors()

	if m.search.pendingSources() == nil {
		m.search.running = false
	}

	// Report failures without wiping the results we did get.
	if len(m.search.failures) > 0 {
		fails := m.search.failuresList()
		first := fails[0]
		n := retryNotice(fmt.Sprintf("%s unavailable", first.Source), first.Err)
		n.detail = hintForSource(first.Source, first.Err)
		if len(fails) > 1 {
			n.text = fmt.Sprintf("%d sources unavailable: %s", len(fails), first.Source)
		}
		if m.search.total() > 0 {
			n.detail = fmt.Sprintf("%d results from other sources · %s", m.search.total(), hintForSource(first.Source, first.Err))
			n.sticky = false
			n.ttl = 10 * time.Second
		}
		m.notice = n
	} else if m.search.total() > 0 {
		m.notice = notice{}
	}

	// Thumbnails are fetched lazily for the selected result.
	next, cmd := m.maybeFetchThumb()
	if next.thumbs != nil {
		m.thumbs = next.thumbs
	}
	m.thumbURL = next.thumbURL
	return cmd
}

// sourceNames lists every configured source name.
func (m Model) sourceNames() []string {
	if len(m.sources) == 0 {
		return search.SourceOrder()
	}
	out := make([]string, 0, len(m.sources))
	for _, s := range m.sources {
		out = append(out, s.Name)
	}
	return out
}

// activeSourceNames lists only the sources this query will actually hit.
//
// Marking every source as pending while the query is restricted to one would
// leave the spinner running forever: the other sources never answer.
func (m Model) activeSourceNames() []string {
	want := m.activeSource()
	if want == "" {
		return m.sourceNames()
	}
	all := m.sourceNames()
	out := make([]string, 0, 1)
	for _, name := range all {
		if canonical, ok := search.CanonicalSource(name); ok && canonical == want {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		// The restricted source is not registered: still expect it, otherwise
		// the search would look finished before it even starts.
		return []string{want}
	}
	return out
}

// ── playing ─────────────────────────────────────────────────────────────────

func (m Model) onPlayOK(msg playOKMsg) (tea.Model, tea.Cmd) {
	if msg.token != m.playToken {
		return m, nil // a newer selection won
	}
	m.err = nil
	m.pendingPlay = models.Song{}
	if msg.song.Title == "" && msg.song.ID == "" {
		m.notice = infoNotice("end of queue", "add something with a or A", "")
		return m, nil
	}

	m.lastSong = msg.song
	if m.hist != nil {
		m.hist.Push(msg.song)
	}
	m.notice = okNotice("playing "+msg.song.DisplayTitle(), "", "")

	// Persist on every track change (not per frame).
	m.saveSession()
	return m, nil
}

// resolveAndPlay returns a Cmd that resolves the song to a stream and starts
// playback. Every message it produces carries the current play token.
func (m Model) resolveAndPlay(song models.Song) tea.Cmd {
	yt := m.youtubeClient
	nd := m.navidromeClient
	p := m.player
	video := m.videoMode
	token := m.playToken

	m.pendingPlay = song

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if p == nil {
			return playErrMsg{err: fmt.Errorf("mpv is not available"), token: token}
		}

		// Video mode: play YouTube results in a real mpv window.
		if video && song.Source == models.SourceYouTube {
			url := song.URL
			if url == "" {
				if song.SourceID == "" {
					return playErrMsg{err: fmt.Errorf("no YouTube id"), token: token}
				}
				url = "https://www.youtube.com/watch?v=" + song.SourceID
			}
			_ = p.Stop()
			if err := p.PlayVideo(url, song.Title); err != nil {
				return playErrMsg{err: fmt.Errorf("video: %w", err), token: token}
			}
			return playOKMsg{song: song, token: token}
		}

		var streamURL string
		var err error

		switch song.Source {
		case models.SourceYouTube:
			if yt == nil {
				return playErrMsg{err: fmt.Errorf("YouTube client unavailable"), token: token}
			}
			streamURL, err = yt.ResolveSongContext(ctx, song)
			if err != nil {
				return playErrMsg{err: fmt.Errorf("YouTube: %w", err), token: token}
			}
		case models.SourceRadio:
			streamURL = song.URL
		case models.SourceNavidrome:
			if nd == nil {
				return playErrMsg{err: fmt.Errorf("Navidrome is not configured (set navidrome_url/user/pass)"), token: token}
			}
			streamURL = nd.StreamURL(song.SourceID)
		case models.SourceLocal:
			streamURL = song.FilePath
			if streamURL == "" {
				streamURL = song.URL
			}
		default:
			streamURL = song.URL
			if streamURL == "" {
				streamURL = song.FilePath
			}
		}

		if streamURL == "" {
			return playErrMsg{
				err:   fmt.Errorf("no playable URL for %q", song.DisplayTitle()),
				token: token,
			}
		}

		if err := p.Play(streamURL, song.Title, song.Artist, song.Album, string(song.Source)); err != nil {
			return playErrMsg{err: err, token: token}
		}
		return playOKMsg{song: song, token: token}
	}
}

// playSelected plays the highlighted result and queues it.
func (m *Model) playSelected() tea.Cmd {
	song := m.selectedSong()
	if song == nil {
		return nil
	}
	m.playToken++
	m.queue.Add(*song)
	m.queue.SetCursor(m.queue.Len() - 1)
	m.focus = FocusList
	return m.resolveAndPlay(*song)
}

func (m *Model) autoAdvance() tea.Cmd {
	// Only a natural end-of-file reaches here: a user-initiated stop does not
	// call OnEnded, so pressing stop can never skip to the next track.
	return m.playNext()
}

// playNext advances the queue and plays the new track.
//
// It takes a pointer receiver on purpose: assigning the notice on a copy (the
// old value-receiver version) silently discarded "end of queue", so the user
// pressed n and nothing at all happened.
func (m *Model) playNext() tea.Cmd {
	if m.queue == nil {
		return nil
	}
	item := m.queue.Next()
	if item == nil {
		m.notice = infoNotice("end of queue", "add songs with a or A", "")
		return nil
	}
	m.playToken++
	cmd := m.resolveAndPlay(item.Song)
	return cmd
}

// prevRestartThreshold is how far into a track "previous" restarts it instead
// of going to the track before — the behaviour every music player has.
const prevRestartThreshold = 3 * time.Second

// playPrev implements the familiar "previous" behaviour: past the first few
// seconds it restarts the current track, otherwise it steps back in the queue.
func (m *Model) playPrev() tea.Cmd {
	if m.queue == nil {
		return nil
	}

	// Restart the current track when it is under way.
	if m.player != nil {
		st := m.player.Status()
		if st.State == player.StatePlaying && st.Elapsed > prevRestartThreshold.Seconds() {
			m.player.SeekAbsolute(0)
			m.notice = okNotice("restarted "+st.Song.Title, "", "")
			return nil
		}
	}

	item := m.queue.Prev()
	if item == nil {
		m.notice = infoNotice("start of queue", "nothing before this track", "")
		return nil
	}
	m.playToken++
	return m.resolveAndPlay(item.Song)
}

// playFromPanel plays the item highlighted in an overlay panel.
//
// Focus moves to the results list: the user just navigated and chose
// something, so single-key actions (n, p, space, a…) must not silently turn
// into typed characters.
func (m *Model) playFromPanel(song models.Song) tea.Cmd {
	m.playToken++
	m.queue.Add(song)
	m.queue.SetCursor(m.queue.Len() - 1)
	m.panel = PanelNone
	m.focus = FocusList
	return m.resolveAndPlay(song)
}

// ── searching ───────────────────────────────────────────────────────────────

// triggerSearch applies the current query and starts a debounced search.
func (m *Model) triggerSearch() {
	src, term := search.ParseQuery(m.search.query)
	m.search.term = term
	m.search.source = src

	m.search.cursor = 0
	m.search.group = 0
	m.err = nil

	if term == "" {
		m.search.reset()
		m.search.term = ""
		if m.sm != nil {
			m.sm.Cancel()
		}
		m.notice = notice{}
		return
	}

	m.search.reset()
	m.search.term = term
	m.search.running = true
	m.search.lastQuery = term
	for _, s := range m.activeSourceNames() {
		m.search.pending[s] = true
	}
	m.notice = notice{}

	if m.sm != nil {
		// Restriction comes from the /prefix or the selector; the query handed
		// to the sources is the term WITHOUT the prefix. Sending the raw query
		// made every source search for the literal "/local wav", and every
		// result then looked like a mismatched query and was dropped.
		switch {
		case src != "":
			m.sm.RestrictTo([]string{src})
		case m.search.restrict != "" && m.search.restrict != "all":
			m.sm.RestrictTo([]string{m.sourceLabel(m.search.restrict)})
		default:
			m.sm.RestrictTo(nil)
		}
		m.sm.Search(term)
	}
}

// retrySearch re-runs the last query immediately (no debounce).
func (m Model) retrySearch() tea.Cmd {
	if m.sm == nil || m.search.lastQuery == "" {
		return nil
	}
	m.search.reset()
	m.search.running = true
	for _, s := range m.activeSourceNames() {
		m.search.pending[s] = true
	}
	m.notice = infoNotice("retrying "+m.search.lastQuery, "", "")
	m.sm.SearchNow(m.search.lastQuery)
	return m.waitSearchEvents()
}

// ── library ─────────────────────────────────────────────────────────────────

// startScan kicks off a library scan in the background.
func (m *Model) startScan() tea.Cmd {
	if m.libraryScanner == nil {
		return nil
	}
	if m.libraryScanner.Scanning() {
		return nil
	}
	ch := m.scanCh
	go func() {
		m.libraryScanner.ScanWithContext(context.Background(), func(st library.ScanStatus) {
			select {
			case ch <- st:
			default:
			}
		})
	}()
	return nil
}

func (m *Model) onScan(st library.ScanStatus) {
	m.lib.status = scanStatusText(st)
	if !st.Running && m.libraryScanner != nil {
		m.lib.sections = m.libraryScanner.Groups()
		if n := m.libraryScanner.Len(); n > 0 {
			m.notice = okNotice(fmt.Sprintf("library indexed · %d tracks", n), "", "")
		}
	}
}

func scanStatusText(st library.ScanStatus) string {
	switch {
	case st.Running:
		return fmt.Sprintf("scanning… %d tracks found", st.Found)
	case st.Err != nil:
		return "scan failed: " + errText(st.Err)
	case st.Found > 0:
		return fmt.Sprintf("%d tracks · scanned %d/%d folders", st.Found, st.Scanned, st.Dirs)
	default:
		return "library is empty"
	}
}

// ── session ─────────────────────────────────────────────────────────────────

// SaveSessionNow persists the session on demand (used on shutdown).
func (m Model) SaveSessionNow() {
	m.saveSession()
}

// saveSession persists the session. It is never called per frame: only on
// track change, on quit and every sessionSaveInterval.
func (m *Model) saveSession() {
	if m.session == nil || !m.restoreSessionOn {
		return
	}
	songs, cursor, shuffle, repeat := m.queue.Snapshot()

	song := m.lastSong
	position := 0.0
	vol := 70
	if m.player != nil {
		st := m.player.Status()
		if song.Title == "" && !st.Song.Empty() {
			song = models.Song{
				Title:  st.Song.Title,
				Artist: st.Song.Artist,
				Album:  st.Song.Album,
				Source: models.SourceType(st.Song.Source),
				URL:    st.Song.URL,
			}
		}
		position = st.Elapsed
		vol = m.player.Volume()
	}

	m.lastSessionAt = time.Now()
	_ = m.session.Save(store.Session{
		Song:     song,
		Position: position,
		Queue:    songs,
		Cursor:   cursor,
		Volume:   vol,
		Shuffle:  shuffle,
		Repeat:   int(repeat),
	})
}

// restoreSession re-applies a stored session: the queue, the playback
// preferences and the last track (which is queued so it can be resumed).
//
// It never starts audio on its own — the user presses enter or n.
func (m *Model) restoreSession() {
	// The flag is the single switch: with restore_session off we neither read
	// nor write the file.
	if m.session == nil || !m.restoreSessionOn {
		return
	}
	sess := m.session.Load()
	if sess.Empty() {
		return
	}

	songs := append([]models.Song{}, sess.Queue...)
	cursor := sess.Cursor

	// The last track may not have been in the queue (played straight from the
	// search results): append it and point the cursor at it.
	if sess.HasSong() && !containsSong(songs, sess.Song) {
		songs = append(songs, sess.Song)
		cursor = len(songs) - 1
	}

	m.queue.Restore(songs, cursor, sess.Shuffle, queue.RepeatMode(sess.Repeat))
	if m.player != nil && sess.Volume > 0 {
		m.player.SetVolume(sess.Volume)
	}
	if sess.HasSong() {
		m.lastSong = sess.Song
	}

	n := infoNotice(
		fmt.Sprintf("session restored · %d tracks", m.queue.Len()),
		"press enter or n to play",
		"[d] clear",
	)
	n.retryable = true // enables the d shortcut above
	n.sticky = false
	n.ttl = 12 * time.Second
	m.notice = n
}

func containsSong(songs []models.Song, target models.Song) bool {
	key := target.Key()
	for _, s := range songs {
		if s.Key() == key {
			return true
		}
	}
	return false
}

// clearSession discards the stored session and empties the queue.
func (m *Model) clearSession() {
	if m.session != nil {
		_ = m.session.Clear()
	}
	m.queue.Clear()
	m.panelCursor = 0
	m.notice = okNotice("session cleared", "the queue is empty", "")
}

// ── helpers ─────────────────────────────────────────────────────────────────

// selectedSong returns the highlighted search result, or nil.
func (m Model) selectedSong() *models.Song {
	groups := m.search.groups()
	if m.search.group < 0 || m.search.group >= len(groups) {
		return nil
	}
	items := groups[m.search.group].Items
	if m.search.cursor < 0 || m.search.cursor >= len(items) {
		return nil
	}
	return &items[m.search.cursor]
}

// currentSong returns the best available "what is playing" song.
func (m Model) currentSong() models.Song {
	if m.lastSong.Title != "" {
		return m.lastSong
	}
	if m.player != nil {
		st := m.player.Status()
		if !st.Song.Empty() {
			return models.Song{
				Title:  st.Song.Title,
				Artist: st.Song.Artist,
				Album:  st.Song.Album,
				Source: models.SourceType(st.Song.Source),
				URL:    st.Song.URL,
			}
		}
	}
	return models.Song{}
}

func (m *Model) clampCursors() {
	groups := m.search.groups()
	if m.search.group >= len(groups) {
		m.search.group = 0
	}
	if m.search.group < 0 {
		m.search.group = 0
	}
	n := 0
	if len(groups) > 0 {
		n = len(groups[m.search.group].Items)
	}
	if m.search.cursor >= n {
		m.search.cursor = 0
	}
	if m.search.cursor < 0 {
		m.search.cursor = 0
	}

	if m.panelCursor >= m.panelLen() {
		m.panelCursor = 0
	}
	if m.panelCursor < 0 {
		m.panelCursor = 0
	}
	if m.panel == PanelPlaylists && m.playls != nil {
		if m.plIndex >= m.playls.Len() {
			m.plIndex = 0
		}
		if m.plIndex < 0 {
			m.plIndex = 0
		}
	}
	if m.lib.cursor >= m.lib.len() {
		m.lib.cursor = 0
	}
	if m.lib.cursor < 0 {
		m.lib.cursor = 0
	}
}

func (m *Model) moveCursor(delta int) {
	groups := m.search.groups()
	if len(groups) == 0 {
		return
	}
	items := groups[m.search.group].Items
	if len(items) == 0 {
		return
	}
	m.search.cursor += delta
	if m.search.cursor < 0 {
		if m.search.group > 0 {
			m.search.group--
			m.search.cursor = len(groups[m.search.group].Items) - 1
		} else {
			m.search.cursor = 0
		}
	}
	if m.search.cursor >= len(items) {
		if m.search.group < len(groups)-1 {
			m.search.group++
			m.search.cursor = 0
		} else {
			m.search.cursor = len(items) - 1
		}
	}
	m.clampCursors()
}

// toggleFavorite flips favourite state for the selected result, or the
// currently playing song when there is no selection.
func (m *Model) toggleFavorite() {
	song := m.selectedSong()
	if song == nil {
		song = m.currentSongPtr()
	}
	if song == nil || song.Title == "" {
		m.notice = infoNotice("nothing to favorite", "select a result first", "")
		return
	}
	if m.favs.Toggle(*song) {
		m.notice = okNotice("★ "+song.DisplayTitle()+" added to favorites", "", "")
	} else {
		m.notice = okNotice("☆ removed from favorites", "", "")
	}
}

func (m Model) currentSongPtr() *models.Song {
	s := m.currentSong()
	if s.Title == "" {
		return nil
	}
	return &s
}

// enqueue adds the selected result without playing it.
func (m *Model) enqueueSelected() {
	song := m.selectedSong()
	if song == nil {
		m.notice = infoNotice("nothing to enqueue", "search something first", "")
		return
	}
	m.queue.Add(*song)
	m.notice = okNotice(fmt.Sprintf("queued %s (%d in queue)", song.DisplayTitle(), m.queue.Len()), "", "")
}

// enqueueAll adds every result currently displayed.
func (m *Model) enqueueAll() {
	groups := m.search.groups()
	n := 0
	for _, g := range groups {
		for _, song := range g.Items {
			if m.queue.AddUnique(song) {
				n++
			}
		}
	}
	if n == 0 {
		if len(groups) == 0 {
			m.notice = infoNotice("nothing to enqueue", "search something first", "")
			return
		}
		m.notice = okNotice("everything is already queued", "", "")
		return
	}
	m.notice = okNotice(fmt.Sprintf("queued %d tracks (%d total)", n, m.queue.Len()), "", "")
}

func (m *Model) cycleRepeat() {
	switch m.queue.Repeat() {
	case queue.RepeatOff:
		m.queue.SetRepeat(queue.RepeatAll)
		m.notice = okNotice("repeat all", "", "")
	case queue.RepeatAll:
		m.queue.SetRepeat(queue.RepeatOne)
		m.notice = okNotice("repeat one", "", "")
	default:
		m.queue.SetRepeat(queue.RepeatOff)
		m.notice = okNotice("repeat off", "", "")
	}
}

func (m *Model) toggleShuffle() {
	on := !m.queue.Shuffle()
	m.queue.SetShuffle(on)
	if on {
		m.notice = okNotice("shuffle on", "", "")
	} else {
		m.notice = okNotice("shuffle off", "", "")
	}
}

func (m *Model) toggleTheme() {
	m.themeIdx = (m.themeIdx + 1) % len(themes)
	m.styles = NewStyles(themes[m.themeIdx])
	m.notice = okNotice("theme: "+themes[m.themeIdx].Name, "", "")
}

func (m *Model) toggleVideo() {
	m.videoMode = !m.videoMode
	if m.videoMode {
		m.notice = warnNotice("video mode on", "YouTube plays in an mpv window (needs yt-dlp)", "esc")
	} else {
		m.notice = okNotice("video mode off", "", "")
	}
}

func (m *Model) changeVolume(delta int) {
	if m.player == nil {
		return
	}
	vol := m.player.Volume() + delta
	m.player.SetVolume(vol)
	m.notice = okNotice(fmt.Sprintf("volume %d%%", m.player.Volume()), "", "")
}

func (m *Model) toggleMute() {
	if m.player == nil {
		return
	}
	m.player.ToggleMute()
	m.notice = okNotice(fmt.Sprintf("volume %d%%", m.player.Volume()), "", "")
}

// showDetails opens the details overlay for the selected (or current) song.
func (m *Model) showDetails() {
	song := m.selectedSong()
	if song == nil {
		song = m.currentSongPtr()
	}
	if song == nil || song.Title == "" {
		m.notice = infoNotice("no track to inspect", "select a result first", "")
		return
	}
	m.details = *song
	m.panel = PanelDetails
	m.panelCursor = 0
}

// removePlayingFromQueue drops the playing track from the queue.
func (m *Model) removePlayingFromQueue() {
	cur := m.queue.Cursor()
	if cur < 0 {
		m.notice = infoNotice("queue is empty", "add songs with a", "")
		return
	}
	title := ""
	if it := m.queue.Current(); it != nil {
		title = it.Song.DisplayTitle()
	}
	m.queue.Remove(cur)
	m.notice = okNotice("removed "+title+" from queue", "", "")
}

// clearQueue empties the queue.
func (m *Model) clearQueue() {
	if m.queue.Len() == 0 {
		m.notice = infoNotice("queue is already empty", "", "")
		return
	}
	n := m.queue.Len()
	m.queue.Clear()
	if m.panel == PanelQueue {
		m.panel = PanelNone
	}
	m.panelCursor = 0
	m.notice = okNotice(fmt.Sprintf("queue cleared (%d tracks)", n), "", "")
}

// cycleSource advances the source selector and restricts the search.
func (m *Model) cycleSource(delta int) {
	if len(m.sources) == 0 {
		return
	}
	names := make([]string, 0, len(m.sources)+1)
	names = append(names, "all")
	for _, s := range m.sources {
		names = append(names, sourceAlias(s.Name))
	}
	idx := 0
	for i, n := range names {
		if n == m.search.restrict {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(names)) % len(names)
	m.search.restrict = names[idx]

	if m.sm != nil && m.search.restrict != "all" {
		m.sm.RestrictTo([]string{m.sourceLabel(m.search.restrict)})
	} else if m.sm != nil {
		m.sm.RestrictTo(nil)
	}
	m.notice = okNotice("source: "+m.sourceLabel(m.search.restrict), "", "")
}

func (m Model) sourceLabel(key string) string {
	if key == "all" {
		return "all"
	}
	for _, s := range m.sources {
		if strings.ToLower(s.Name) == key {
			return s.Name
		}
	}
	return key
}

// activeSource is the canonical source currently in effect: the /prefix typed in
// the query wins over the selector, because it is the more explicit of the two.
func (m Model) activeSource() string {
	if m.search.source != "" {
		return m.search.source
	}
	if canonical, ok := search.CanonicalSource(m.search.restrict); ok {
		return canonical
	}
	return ""
}

func (m Model) panelLen() int {
	switch m.panel {
	case PanelQueue:
		return m.queue.Len()
	case PanelFavorites:
		return m.favs.Len()
	case PanelHistory:
		return m.hist.Len()
	case PanelPlaylists:
		if m.plLevel == 1 {
			if pl := m.currentPlaylist(); pl != nil {
				return len(pl.Tracks)
			}
			return 0
		}
		if m.playls != nil {
			return m.playls.Len()
		}
		return 0
	case PanelLibrary:
		return m.lib.len()
	case PanelDetails:
		return 0
	default:
		return 0
	}
}

func isPrintable(key string) bool {
	r := []rune(key)
	if len(r) != 1 {
		return false
	}
	return unicode.IsPrint(r[0]) && !unicode.IsControl(r[0])
}
