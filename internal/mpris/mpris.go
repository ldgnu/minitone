// Package mpris exposes minitone on the D-Bus session bus following the MPRIS
// specification (org.mpris.MediaPlayer2), so launchers and playerctl can see
// and control it.
//
// It is a thin, read-only-from-state bridge: it mirrors the live player status
// (PlaybackStatus, Metadata, Volume, Position) and forwards transport controls
// (Play/Pause/Stop/Seek/SetVolume) straight to the *player.Player, while
// Next/Previous are routed to the UI queue through channels (the same path the
// keyboard shortcuts use). Nothing here starts mpv or changes how minitone
// already plays; it only publishes what the player already knows.
package mpris

import (
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/ldgnu/minitone/internal/events"
	"github.com/ldgnu/minitone/internal/player"
)

const (
	busName    = "org.mpris.MediaPlayer2.minitone"
	objectPath = "/org/mpris/MediaPlayer2"
	rootIface  = "org.mpris.MediaPlayer2"
	playerIface = "org.mpris.MediaPlayer2.Player"
	propsIface = "org.freedesktop.DBus.Properties"
)

// Service is the D-Bus MPRIS endpoint for a single minitone player.
type Service struct {
	p      *player.Player
	nextCh chan struct{}
	prevCh chan struct{}

	conn *dbus.Conn
	done chan struct{}
	evCh chan struct{}

	mu    sync.Mutex
	running bool

	// Position tracking: the player reports Elapsed only on mpv property
	// updates, so we anchor the last known value with a timestamp and extrapolate
	// while playing to keep Position accurate between polls.
	posBase  float64
	posStamp time.Time

	trackID          uint64
	lastPlayback     string
	lastMetaHash     string
	lastVolume       float64
	lastPosSec       int64
}

// New builds an MPRIS service bound to p. nextCh/prevCh receive a token whenever
// a D-Bus Next/Previous is requested so the UI can advance/rewind the queue.
func New(p *player.Player, nextCh, prevCh chan struct{}) *Service {
	return &Service{
		p:      p,
		nextCh: nextCh,
		prevCh: prevCh,
		done:   make(chan struct{}),
		evCh:   make(chan struct{}, 32),
	}
}

// Start connects to the session bus, claims the stable bus name and exports the
// MPRIS interfaces. It returns an error instead of crashing the app when there is
// no session bus (headless/CI): minitone stays fully usable from the terminal.
func (s *Service) Start() error {
	if s.p == nil {
		return fmt.Errorf("no player to expose")
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}

	reply, err := conn.RequestName(busName,
		dbus.NameFlagReplaceExisting|dbus.NameFlagAllowReplacement)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("request name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return fmt.Errorf("bus name %s already owned (reply %d)", busName, reply)
	}

	root := &rootObj{svc: s}
	pl := &playerObj{svc: s}
	props := &propsObj{svc: s}
	intro := &introObj{svc: s}

	for iface, obj := range map[string]interface{}{
		rootIface:   root,
		playerIface: pl,
		propsIface:  props,
		"org.freedesktop.DBus.Introspectable": intro,
	} {
		if err := conn.Export(obj, objectPath, iface); err != nil {
			_ = conn.Close()
			return fmt.Errorf("export %s: %w", iface, err)
		}
	}

	// ExportMethodTable lets us register the D-Bus member "Seek" (required by
	// the MPRIS spec) under a Go function whose name is not Seek, avoiding the
	// go vet stdmethods false positive that would otherwise treat it as an
	// io.Seeker. All other members are exported the normal way above.
	playerMethods := map[string]any{
		"Seek": s.mSeek,
	}
	if err := conn.ExportMethodTable(playerMethods, objectPath, playerIface); err != nil {
		_ = conn.Close()
		return fmt.Errorf("export Seek: %w", err)
	}

	s.conn = conn
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// Prompt updates on player events without blocking the player's own
	// goroutine: handlers just push a token onto a buffered channel.
	ev := events.Global()
	handler := func(events.Event) {
		select {
		case s.evCh <- struct{}{}:
		default:
		}
	}
	ev.Subscribe(events.EventSongPlayed, handler)
	ev.Subscribe(events.EventSongPaused, handler)
	ev.Subscribe(events.EventSongResumed, handler)
	ev.Subscribe(events.EventSongStopped, handler)
	ev.Subscribe(events.EventVolumeChanged, handler)

	go s.watch()
	return nil
}

// Stop releases the bus name and closes the connection. Safe to call once.
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	conn := s.conn
	s.mu.Unlock()

	close(s.done)
	if conn != nil {
		_, _ = conn.ReleaseName(busName)
		_ = conn.Close()
	}
}

// watch pumps property changes: on every player event and on a steady ticker so
// Position stays fresh while playing.
func (s *Service) watch() {
	s.publishChanges() // seed initial state
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-s.evCh:
		case <-ticker.C:
		}
		s.publishChanges()
	}
}

// ── state helpers ────────────────────────────────────────────────────────────

func (s *Service) playbackStatus() string {
	switch s.p.Status().State {
	case player.StatePlaying:
		return "Playing"
	case player.StatePaused:
		return "Paused"
	default:
		return "Stopped"
	}
}

func (s *Service) volume() float64 {
	return float64(s.p.Volume()) / 100.0
}

func (s *Service) positionMicros() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos := s.posBase
	if s.lastPlayback == "Playing" {
		pos += time.Since(s.posStamp).Seconds()
	}
	return int64(pos * 1e6)
}

func metaHash(song player.SongInfo) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s", song.Title, song.Artist, song.Album, song.URL)
}

func (s *Service) metadata() map[string]dbus.Variant {
	st := s.p.Status()
	song := st.Song

	title := song.Title
	if title == "" {
		title = "minitone"
	}
	artists := []string{}
	if song.Artist != "" {
		artists = append(artists, song.Artist)
	}
	if len(artists) == 0 {
		artists = append(artists, "Unknown Artist")
	}

	meta := map[string]dbus.Variant{
		"xesam:title":  dbus.MakeVariant(title),
		"xesam:artist": dbus.MakeVariant(artists),
		"xesam:album":  dbus.MakeVariant(song.Album),
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath(
			fmt.Sprintf("/org/mpris/MediaPlayer2/Track/%d", s.trackID))),
		"mpris:length": dbus.MakeVariant(int64(st.Duration * 1e6)),
	}
	if song.URL != "" {
		meta["xesam:url"] = dbus.MakeVariant(song.URL)
	}
	return meta
}

// publishChanges recomputes the snapshot and emits PropertiesChanged for whatever
// actually moved since the last publish.
func (s *Service) publishChanges() {
	st := s.p.Status()

	s.mu.Lock()
	s.posBase = st.Elapsed
	s.posStamp = time.Now()
	s.mu.Unlock()

	ps := s.playbackStatus()
	changed := map[string]dbus.Variant{}

	if ps != s.lastPlayback {
		s.lastPlayback = ps
		changed["PlaybackStatus"] = dbus.MakeVariant(ps)
	}

	hash := metaHash(st.Song)
	if hash != s.lastMetaHash {
		s.lastMetaHash = hash
		s.trackID++
		if s.trackID == 0 {
			s.trackID = 1
		}
		changed["Metadata"] = dbus.MakeVariant(s.metadata())
		// A track change resets the playhead.
		s.mu.Lock()
		s.posBase = 0
		s.posStamp = time.Now()
		s.mu.Unlock()
		changed["Position"] = dbus.MakeVariant(int64(0))
	}

	vol := s.volume()
	if vol != s.lastVolume {
		s.lastVolume = vol
		changed["Volume"] = dbus.MakeVariant(vol)
	}

	if ps == "Playing" {
		pos := s.positionMicros()
		if pos/1e6 != s.lastPosSec {
			s.lastPosSec = pos / 1e6
			changed["Position"] = dbus.MakeVariant(pos)
		}
	}

	if len(changed) > 0 {
		s.emitPropsChanged(changed)
	}
}

func (s *Service) emitPropsChanged(changed map[string]dbus.Variant) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Emit(objectPath, propsIface+".PropertiesChanged",
		playerIface, changed, []string{})
}

// ── transport controls ───────────────────────────────────────────────────────

func (s *Service) cmdPlay() {
	switch s.p.Status().State {
	case player.StatePaused:
		s.p.TogglePause()
	case player.StateStopped:
		s.requestNext() // nothing is loaded: start the queue
	}
}

func (s *Service) cmdPause() {
	if s.p.Status().State == player.StatePlaying {
		s.p.TogglePause()
	}
}

func (s *Service) requestNext() {
	if s.nextCh != nil {
		select {
		case s.nextCh <- struct{}{}:
		default:
		}
	}
}

func (s *Service) requestPrev() {
	if s.prevCh != nil {
		select {
		case s.prevCh <- struct{}{}:
		default:
		}
	}
}

// ── D-Bus object handlers ──────────────────────────────────────────────────
// Separate structs per interface keep introspection honest: each interface
// exports only its own members.

// mSeek implements the MPRIS "Seek" member. It is named mSeek (not Seek) so go
// vet's stdmethods analyzer does not mistake it for io.Seeker; it is exposed as
// "Seek" via ExportMethodTable in Start.
func (s *Service) mSeek(offset int64) *dbus.Error {
	s.p.SeekAbsolute(float64(offset) / 1e6)
	s.emitSeeked(s.positionMicros())
	return nil
}

func (s *Service) emitSeeked(pos int64) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Emit(objectPath, playerIface+".Seeked", pos)
}

type rootObj struct{ svc *Service }

func (o *rootObj) Raise() *dbus.Error { return nil }
func (o *rootObj) Quit() *dbus.Error  { return nil }

type playerObj struct{ svc *Service }

func (o *playerObj) PlayPause() *dbus.Error           { o.svc.p.TogglePause(); return nil }
func (o *playerObj) Play() *dbus.Error                { o.svc.cmdPlay(); return nil }
func (o *playerObj) Pause() *dbus.Error               { o.svc.cmdPause(); return nil }
func (o *playerObj) Stop() *dbus.Error                { _ = o.svc.p.Stop(); return nil }
func (o *playerObj) Next() *dbus.Error                { o.svc.requestNext(); return nil }
func (o *playerObj) Previous() *dbus.Error            { o.svc.requestPrev(); return nil }
func (o *playerObj) SetPosition(t dbus.ObjectPath, pos int64) *dbus.Error {
	o.svc.p.SeekAbsolute(float64(pos) / 1e6)
	o.svc.emitSeeked(pos)
	return nil
}
func (o *playerObj) OpenUri(uri string) *dbus.Error { return nil }

type propsObj struct{ svc *Service }

func (o *propsObj) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	v, err := o.svc.getProperty(iface, prop)
	if err != nil {
		return dbus.Variant{}, dbus.MakeFailedError(err)
	}
	return v, nil
}

func (o *propsObj) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	m, err := o.svc.getAllProperties(iface)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}
	return m, nil
}

func (o *propsObj) Set(iface, prop string, val dbus.Variant) *dbus.Error {
	if err := o.svc.setProperty(iface, prop, val); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

type introObj struct{ svc *Service }

func (o *introObj) Introspect() (string, *dbus.Error) {
	return introspectionXML, nil
}

// ── property access ──────────────────────────────────────────────────────────

func (s *Service) getProperty(iface, prop string) (dbus.Variant, error) {
	switch iface {
	case rootIface:
		switch prop {
		case "Identity":
			return dbus.MakeVariant("minitone"), nil
		case "HasTrackList":
			return dbus.MakeVariant(false), nil
		case "SupportedUriSchemes":
			return dbus.MakeVariant([]string{"file", "http", "https", "youtube", "radio"}), nil
		case "SupportedMimeTypes":
			return dbus.MakeVariant([]string{
				"audio/mpeg", "audio/flac", "audio/ogg",
				"audio/aac", "audio/x-m4a", "audio/webm",
			}), nil
		}
	case playerIface:
		switch prop {
		case "PlaybackStatus":
			return dbus.MakeVariant(s.playbackStatus()), nil
		case "Metadata":
			return dbus.MakeVariant(s.metadata()), nil
		case "Volume":
			return dbus.MakeVariant(s.volume()), nil
		case "Position":
			return dbus.MakeVariant(s.positionMicros()), nil
		case "Rate", "MinimumRate", "MaximumRate":
			return dbus.MakeVariant(1.0), nil
		case "LoopStatus":
			return dbus.MakeVariant("None"), nil
		case "CanControl", "CanPlay", "CanPause", "CanSeek",
			"CanGoNext", "CanGoPrevious":
			return dbus.MakeVariant(true), nil
		}
	}
	return dbus.Variant{}, fmt.Errorf("no such property %s.%s", iface, prop)
}

func (s *Service) getAllProperties(iface string) (map[string]dbus.Variant, error) {
	out := map[string]dbus.Variant{}
	var props []string
	switch iface {
	case rootIface:
		props = []string{"Identity", "HasTrackList", "SupportedUriSchemes", "SupportedMimeTypes"}
	case playerIface:
		props = []string{
			"PlaybackStatus", "LoopStatus", "Rate", "Metadata", "Volume",
			"Position", "MinimumRate", "MaximumRate",
			"CanGoNext", "CanGoPrevious", "CanPlay", "CanPause", "CanSeek", "CanControl",
		}
	default:
		return nil, fmt.Errorf("no such interface %s", iface)
	}
	for _, p := range props {
		v, err := s.getProperty(iface, p)
		if err != nil {
			return nil, err
		}
		out[p] = v
	}
	return out, nil
}

func (s *Service) setProperty(iface, prop string, val dbus.Variant) error {
	if iface != playerIface || prop != "Volume" {
		return fmt.Errorf("property %s.%s is not settable", iface, prop)
	}
	var v float64
	if err := val.Store(&v); err != nil {
		return err
	}
	s.p.SetVolume(int(v * 100))
	return nil
}

const introspectionXML = `<!DOCTYPE node PUBLIC "-//freedesktop//DTD D-BUS Object Introspection 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd">
<node>
  <interface name="org.mpris.MediaPlayer2">
    <method name="Raise"/>
    <method name="Quit"/>
    <property name="Identity" type="s" access="read"/>
    <property name="HasTrackList" type="b" access="read"/>
    <property name="SupportedUriSchemes" type="as" access="read"/>
    <property name="SupportedMimeTypes" type="as" access="read"/>
  </interface>
  <interface name="org.mpris.MediaPlayer2.Player">
    <method name="Next"/>
    <method name="Previous"/>
    <method name="Pause"/>
    <method name="PlayPause"/>
    <method name="Stop"/>
    <method name="Play"/>
    <method name="Seek"><arg name="Offset" type="x" direction="in"/></method>
    <method name="SetPosition"><arg name="TrackId" type="o" direction="in"/><arg name="Position" type="x" direction="in"/></method>
    <method name="OpenUri"><arg name="Uri" type="s" direction="in"/></method>
    <signal name="Seeked"><arg name="Position" type="x"/></signal>
    <property name="PlaybackStatus" type="s" access="read"/>
    <property name="LoopStatus" type="s" access="readwrite"/>
    <property name="Rate" type="d" access="readwrite"/>
    <property name="Metadata" type="a{sv}" access="read"/>
    <property name="Volume" type="d" access="readwrite"/>
    <property name="Position" type="x" access="read"/>
    <property name="MinimumRate" type="d" access="read"/>
    <property name="MaximumRate" type="d" access="read"/>
    <property name="CanGoNext" type="b" access="read"/>
    <property name="CanGoPrevious" type="b" access="read"/>
    <property name="CanPlay" type="b" access="read"/>
    <property name="CanPause" type="b" access="read"/>
    <property name="CanSeek" type="b" access="read"/>
    <property name="CanControl" type="b" access="read"/>
  </interface>
  <interface name="org.freedesktop.DBus.Properties">
    <method name="Get"><arg name="interface_name" type="s" direction="in"/><arg name="property_name" type="s" direction="in"/><arg name="value" type="v" direction="out"/></method>
    <method name="Set"><arg name="interface_name" type="s" direction="in"/><arg name="property_name" type="s" direction="in"/><arg name="value" type="v" direction="in"/></method>
    <method name="GetAll"><arg name="interface_name" type="s" direction="in"/><arg name="properties" type="a{sv}" direction="out"/></method>
    <signal name="PropertiesChanged"><arg name="interface_name" type="s"/><arg name="changed_properties" type="a{sv}"/><arg name="invalidated_properties" type="as"/></signal>
  </interface>
  <interface name="org.freedesktop.DBus.Introspectable">
    <method name="Introspect"><arg name="data" type="s" direction="out"/></method>
  </interface>
</node>`
