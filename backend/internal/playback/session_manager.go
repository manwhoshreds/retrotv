package playback

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

var ErrInvalidChannelID = errors.New("invalid channel id")

type ChannelRequest struct {
	ChannelID int64
	BaseURL   string
	Request   Request
	ProgramID int64
	StartTime time.Time
	EndTime   time.Time
}

type SessionStarter interface {
	StartSession(ctx context.Context, request ChannelRequest) (SessionProcess, error)
}

type ChannelRequestProvider interface {
	CurrentChannelRequest(ctx context.Context, channelID int64, now time.Time) (ChannelRequest, bool, error)
}

type SessionProcess interface {
	Playback() Playback
	Stop(ctx context.Context) error
	Done() <-chan error
}

type SessionState string

const (
	SessionStarting SessionState = "starting"
	SessionReady    SessionState = "ready"
	SessionStopping SessionState = "stopping"
)

type SessionManager struct {
	starter      SessionStarter
	provider     ChannelRequestProvider
	restartDelay time.Duration
	stopTimeout  time.Duration
	idleTimeout  time.Duration
	logger       *log.Logger
	stopping     chan struct{}
	stopOnce     sync.Once

	mu       sync.Mutex
	sessions map[int64]*managedSession
}

type SessionManagerOption func(*SessionManager)

func WithRestartDelay(delay time.Duration) SessionManagerOption {
	return func(manager *SessionManager) {
		manager.restartDelay = delay
	}
}

func WithStopTimeout(timeout time.Duration) SessionManagerOption {
	return func(manager *SessionManager) {
		manager.stopTimeout = timeout
	}
}

func WithIdleTimeout(timeout time.Duration) SessionManagerOption {
	return func(manager *SessionManager) {
		manager.idleTimeout = timeout
	}
}

func WithLogger(logger *log.Logger) SessionManagerOption {
	return func(manager *SessionManager) {
		manager.logger = logger
	}
}

func WithChannelRequestProvider(provider ChannelRequestProvider) SessionManagerOption {
	return func(manager *SessionManager) {
		manager.provider = provider
	}
}

func NewSessionManager(starter SessionStarter, options ...SessionManagerOption) *SessionManager {
	manager := &SessionManager{
		starter:      starter,
		restartDelay: 2 * time.Second,
		stopTimeout:  5 * time.Second,
		idleTimeout:  5 * time.Minute,
		logger:       log.New(os.Stderr, "", log.LstdFlags),
		stopping:     make(chan struct{}),
		sessions:     map[int64]*managedSession{},
	}
	for _, option := range options {
		option(manager)
	}
	return manager
}

func (m *SessionManager) StartChannels(ctx context.Context, requests []ChannelRequest) error {
	for _, request := range requests {
		if err := m.EnsureChannel(ctx, request); err != nil {
			return err
		}
	}
	return nil
}

func (m *SessionManager) EnsureChannel(ctx context.Context, request ChannelRequest) error {
	if request.ChannelID <= 0 {
		return ErrInvalidChannelID
	}
	if request.Request.ChannelID == 0 {
		request.Request.ChannelID = request.ChannelID
	}

	startedAt := time.Now()
	for {
		m.mu.Lock()
		if session := m.sessions[request.ChannelID]; session != nil {
			session.request = request
			session.lastAccess = time.Now()
			if session.state == SessionReady {
				m.scheduleTransitionLocked(session)
				m.scheduleIdleCleanupLocked(session)
				m.mu.Unlock()
				m.logf("event=stream_reuse channel=%d program=%d state=%s", request.ChannelID, request.ProgramID, SessionReady)
				return nil
			}
			ready := session.ready
			m.mu.Unlock()
			select {
			case <-ready:
				m.mu.Lock()
				err := session.startErr
				m.mu.Unlock()
				if err != nil {
					return err
				}
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		session := &managedSession{
			request:    request,
			running:    true,
			state:      SessionStarting,
			ready:      make(chan struct{}),
			lastAccess: time.Now(),
		}
		m.sessions[request.ChannelID] = session
		m.mu.Unlock()

		m.logf("event=stream_create channel=%d program=%d state=%s", request.ChannelID, request.ProgramID, SessionStarting)
		process, err := m.starter.StartSession(ctx, request)

		m.mu.Lock()
		current := m.sessions[request.ChannelID]
		if current != session {
			m.mu.Unlock()
			m.stopProcess(context.Background(), process)
			return nil
		}
		if err != nil {
			session.running = false
			session.state = SessionStopping
			session.startErr = err
			delete(m.sessions, request.ChannelID)
			close(session.ready)
			m.mu.Unlock()
			m.logf("event=stream_start_failed channel=%d program=%d latency_ms=%d err=%v", request.ChannelID, request.ProgramID, time.Since(startedAt).Milliseconds(), err)
			return err
		}

		session.process = process
		session.playback = process.Playback()
		session.state = SessionReady
		close(session.ready)
		m.scheduleTransitionLocked(session)
		m.scheduleIdleCleanupLocked(session)
		m.mu.Unlock()

		m.logf("event=stream_ready channel=%d program=%d latency_ms=%d url=%q", request.ChannelID, request.ProgramID, time.Since(startedAt).Milliseconds(), session.playback.URL)
		go m.monitorProcess(request.ChannelID, session, process)
		return nil
	}
}

func (m *SessionManager) Touch(channelID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.sessions[channelID]; session != nil {
		session.lastAccess = time.Now()
		m.scheduleIdleCleanupLocked(session)
	}
}

func (m *SessionManager) Playback(channelID int64) (Playback, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[channelID]
	if session == nil || !session.running || session.state != SessionReady {
		return Playback{}, false
	}
	return session.playback, true
}

func (m *SessionManager) State(channelID int64) (SessionState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[channelID]
	if session == nil {
		return "", false
	}
	return session.state, true
}

func (m *SessionManager) ActiveSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *SessionManager) StopChannel(ctx context.Context, channelID int64) {
	m.mu.Lock()
	session := m.sessions[channelID]
	if session != nil {
		session.running = false
		session.state = SessionStopping
		if session.transitionTimer != nil {
			session.transitionTimer.Stop()
			session.transitionTimer = nil
		}
		if session.idleTimer != nil {
			session.idleTimer.Stop()
			session.idleTimer = nil
		}
		delete(m.sessions, channelID)
	}
	m.mu.Unlock()
	if session != nil {
		m.stopProcess(ctx, session.process)
	}
}

// RetainChannels stops every session whose channel is not in the retained set.
// It is used by clients with bounded prewarming so old windows do not accumulate
// FFmpeg processes until the normal idle timeout.
func (m *SessionManager) RetainChannels(ctx context.Context, channelIDs []int64) {
	retained := make(map[int64]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID > 0 {
			retained[channelID] = struct{}{}
		}
	}

	m.mu.Lock()
	removed := make([]*managedSession, 0)
	for channelID, session := range m.sessions {
		if _, ok := retained[channelID]; ok {
			continue
		}
		session.running = false
		session.state = SessionStopping
		if session.transitionTimer != nil {
			session.transitionTimer.Stop()
			session.transitionTimer = nil
		}
		if session.idleTimer != nil {
			session.idleTimer.Stop()
			session.idleTimer = nil
		}
		delete(m.sessions, channelID)
		removed = append(removed, session)
	}
	m.mu.Unlock()

	for _, session := range removed {
		m.stopProcess(ctx, session.process)
	}
}

func (m *SessionManager) StopAll(ctx context.Context) {
	m.stopOnce.Do(func() {
		close(m.stopping)
	})
	m.stopActiveSessions(ctx)
}

func (m *SessionManager) StopActiveSessions(ctx context.Context) {
	m.stopActiveSessions(ctx)
}

func (m *SessionManager) stopActiveSessions(ctx context.Context) {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = map[int64]*managedSession{}
	m.mu.Unlock()

	for _, session := range sessions {
		session.running = false
		session.state = SessionStopping
		if session.transitionTimer != nil {
			session.transitionTimer.Stop()
			session.transitionTimer = nil
		}
		if session.idleTimer != nil {
			session.idleTimer.Stop()
			session.idleTimer = nil
		}
		m.stopProcess(ctx, session.process)
	}
}

func (m *SessionManager) startChannel(ctx context.Context, request ChannelRequest) error {
	return m.EnsureChannel(ctx, request)
}

func (m *SessionManager) monitorProcess(channelID int64, session *managedSession, process SessionProcess) {
	err := <-process.Done()

	m.mu.Lock()
	current := m.sessions[channelID]
	if current != session || session.process != process || !session.running {
		m.mu.Unlock()
		return
	}
	if session.transitionTimer != nil {
		session.transitionTimer.Stop()
		session.transitionTimer = nil
	}
	if session.idleTimer != nil {
		session.idleTimer.Stop()
		session.idleTimer = nil
	}
	delete(m.sessions, channelID)
	request := session.request
	m.mu.Unlock()

	m.logf("event=ffmpeg_exit channel=%d program=%d err=%v", channelID, request.ProgramID, err)

	select {
	case <-m.stopping:
		return
	case <-time.After(m.restartDelay):
	}

	select {
	case <-m.stopping:
		return
	default:
	}
	if err := m.startChannel(context.Background(), request); err != nil {
		m.logf("event=stream_restart_failed channel=%d program=%d err=%v", channelID, request.ProgramID, err)
		go m.restartChannelLater(request)
	}
}

func (m *SessionManager) restartChannelLater(request ChannelRequest) {
	select {
	case <-m.stopping:
		return
	case <-time.After(m.restartDelay):
	}
	select {
	case <-m.stopping:
		return
	default:
	}
	if err := m.startChannel(context.Background(), request); err != nil {
		m.logf("event=stream_restart_retry_failed channel=%d program=%d err=%v", request.ChannelID, request.ProgramID, err)
		go m.restartChannelLater(request)
	}
}

func (m *SessionManager) scheduleTransitionLocked(session *managedSession) {
	if session.transitionTimer != nil {
		session.transitionTimer.Stop()
		session.transitionTimer = nil
	}
	if m.provider == nil || session.request.EndTime.IsZero() {
		return
	}
	if session.state != SessionReady {
		return
	}
	delay := nextTransitionDelay(time.Now().UTC(), session.request.EndTime)
	channelID := session.request.ChannelID
	transitionAt := session.request.EndTime
	session.transitionTimer = time.AfterFunc(delay, func() {
		m.transitionChannel(channelID, session, transitionAt)
	})
}

func nextTransitionDelay(now time.Time, end time.Time) time.Duration {
	delay := end.Sub(now)
	if delay < 0 {
		return 0
	}
	return delay
}

func (m *SessionManager) transitionChannel(channelID int64, session *managedSession, transitionAt time.Time) {
	select {
	case <-m.stopping:
		return
	default:
	}

	m.mu.Lock()
	current := m.sessions[channelID]
	if current != session || !session.running || !session.request.EndTime.Equal(transitionAt) {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	now := time.Now().UTC()
	nextRequest, ok, err := m.provider.CurrentChannelRequest(context.Background(), channelID, now)
	if err != nil {
		m.logf("event=stream_transition_lookup_failed channel=%d expected_transition=%s err=%v", channelID, transitionAt.Format(time.RFC3339), err)
		return
	}
	if !ok {
		m.logf("event=stream_transition_missing_program channel=%d expected_transition=%s", channelID, transitionAt.Format(time.RFC3339))
		m.StopChannel(context.Background(), channelID)
		return
	}
	if nextRequest.ChannelID == 0 {
		nextRequest.ChannelID = channelID
	}

	m.mu.Lock()
	current = m.sessions[channelID]
	if current != session || !session.running {
		m.mu.Unlock()
		return
	}
	oldProcess := session.process
	oldRequest := session.request
	session.process = nil
	session.state = SessionStarting
	m.mu.Unlock()

	m.stopProcess(context.Background(), oldProcess)

	process, err := m.starter.StartSession(context.Background(), nextRequest)
	if err != nil {
		m.logf("event=stream_transition_start_failed channel=%d previous_program=%d next_program=%d err=%v", channelID, oldRequest.ProgramID, nextRequest.ProgramID, err)
		m.mu.Lock()
		if m.sessions[channelID] == session {
			session.running = false
			if session.transitionTimer != nil {
				session.transitionTimer.Stop()
				session.transitionTimer = nil
			}
			if session.idleTimer != nil {
				session.idleTimer.Stop()
				session.idleTimer = nil
			}
			delete(m.sessions, channelID)
		}
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	if m.sessions[channelID] != session || !session.running {
		m.mu.Unlock()
		m.stopProcess(context.Background(), process)
		return
	}
	session.request = nextRequest
	session.process = process
	session.state = SessionReady
	session.playback = stablePlayback(session.playback, process.Playback())
	m.scheduleTransitionLocked(session)
	m.scheduleIdleCleanupLocked(session)
	m.mu.Unlock()

	go m.monitorProcess(channelID, session, process)
}

func (m *SessionManager) scheduleIdleCleanupLocked(session *managedSession) {
	if session.idleTimer != nil {
		session.idleTimer.Stop()
		session.idleTimer = nil
	}
	if m.idleTimeout <= 0 || session.state != SessionReady {
		return
	}
	channelID := session.request.ChannelID
	session.idleTimer = time.AfterFunc(m.idleTimeout, func() {
		m.cleanupIdleChannel(channelID, session)
	})
}

func (m *SessionManager) cleanupIdleChannel(channelID int64, session *managedSession) {
	m.mu.Lock()
	current := m.sessions[channelID]
	if current != session || session.state != SessionReady || time.Since(session.lastAccess) < m.idleTimeout {
		m.mu.Unlock()
		return
	}
	session.running = false
	session.state = SessionStopping
	if session.transitionTimer != nil {
		session.transitionTimer.Stop()
		session.transitionTimer = nil
	}
	if session.idleTimer != nil {
		session.idleTimer.Stop()
		session.idleTimer = nil
	}
	process := session.process
	programID := session.request.ProgramID
	m.mu.Unlock()

	m.logf("event=stream_cleanup channel=%d program=%d reason=idle timeout_ms=%d", channelID, programID, m.idleTimeout.Milliseconds())
	m.stopProcess(context.Background(), process)

	m.mu.Lock()
	if m.sessions[channelID] == session {
		delete(m.sessions, channelID)
	}
	m.mu.Unlock()
}

func stablePlayback(current Playback, next Playback) Playback {
	if current.URL == "" {
		return next
	}
	next.URL = current.URL
	return next
}

func (m *SessionManager) stopProcess(ctx context.Context, process SessionProcess) {
	if process == nil {
		return
	}
	timeout := m.stopTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_ = process.Stop(stopCtx)
}

type managedSession struct {
	request         ChannelRequest
	process         SessionProcess
	playback        Playback
	running         bool
	state           SessionState
	ready           chan struct{}
	startErr        error
	lastAccess      time.Time
	transitionTimer *time.Timer
	idleTimer       *time.Timer
}

func (m *SessionManager) logf(format string, args ...any) {
	if m.logger != nil {
		m.logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}
