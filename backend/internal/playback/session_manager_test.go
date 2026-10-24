package playback

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"retrotv/internal/presentation"
)

func TestSessionManagerStartsPersistentChannels(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithRestartDelay(time.Hour))

	err := manager.StartChannels(context.Background(), []ChannelRequest{
		{ChannelID: 7, Request: Request{Program: Program{MediaType: MediaTypeMovie, MovieID: 42}, SeekPosition: 120}},
		{ChannelID: 8, Request: Request{Program: Program{MediaType: MediaTypeMovie, MovieID: 43}, SeekPosition: 90}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if starter.starts != 2 {
		t.Fatalf("starts = %d, want 2", starter.starts)
	}
	if manager.ActiveSessions() != 2 {
		t.Fatalf("active sessions = %d, want 2", manager.ActiveSessions())
	}
	playback, ok := manager.Playback(7)
	if !ok {
		t.Fatal("channel 7 playback missing")
	}
	if playback.URL != "http://retrotv.test/streams/channel/7" || playback.SeekPosition != 120 {
		t.Fatalf("playback = %#v", playback)
	}
}

func TestSessionManagerEnsureChannelReusesExistingProcess(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithRestartDelay(time.Hour))

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 11}); err != nil {
		t.Fatal(err)
	}
	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 11}); err != nil {
		t.Fatal(err)
	}

	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1", starter.starts)
	}
}

func TestSessionManagerWaitsForStartingSessionAndReusesIt(t *testing.T) {
	starter := &fakeSessionStarter{startBlock: make(chan struct{})}
	manager := NewSessionManager(starter, WithRestartDelay(time.Hour))

	errs := make(chan error, 2)
	go func() {
		errs <- manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 31})
	}()
	waitFor(t, time.Second, func() bool {
		return starter.starts == 1
	})
	go func() {
		errs <- manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 31})
	}()
	time.Sleep(10 * time.Millisecond)
	close(starter.startBlock)

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1", starter.starts)
	}
	state, ok := manager.State(31)
	if !ok || state != SessionReady {
		t.Fatalf("state = %q ok=%v, want ready", state, ok)
	}
}

func TestSessionManagerCleansUpIdleSession(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(
		starter,
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
		WithIdleTimeout(10*time.Millisecond),
	)

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 33}); err != nil {
		t.Fatal(err)
	}
	firstProcess := starter.processes[0]

	waitFor(t, time.Second, func() bool {
		return manager.ActiveSessions() == 0
	})
	if !firstProcess.stopped {
		t.Fatal("idle process was not stopped")
	}
}

func TestSessionManagerTouchKeepsSessionAlive(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(
		starter,
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
		WithIdleTimeout(25*time.Millisecond),
	)

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 34}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		time.Sleep(10 * time.Millisecond)
		manager.Touch(34)
	}
	if manager.ActiveSessions() != 1 {
		t.Fatalf("active sessions = %d, want 1", manager.ActiveSessions())
	}
}

func TestSessionManagerRetainChannelsStopsSessionsOutsideSet(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithStopTimeout(time.Second))
	for _, channelID := range []int64{41, 42, 43} {
		if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: channelID}); err != nil {
			t.Fatal(err)
		}
	}

	manager.RetainChannels(context.Background(), []int64{42})

	if manager.ActiveSessions() != 1 {
		t.Fatalf("active sessions = %d, want 1", manager.ActiveSessions())
	}
	if _, ok := manager.State(42); !ok {
		t.Fatal("retained channel 42 was stopped")
	}
	if !starter.processes[0].stopped || starter.processes[1].stopped || !starter.processes[2].stopped {
		t.Fatalf("stopped states = %t, %t, %t", starter.processes[0].stopped, starter.processes[1].stopped, starter.processes[2].stopped)
	}
}

func TestSessionManagerLogsCreationReuseAndCleanup(t *testing.T) {
	var logs bytes.Buffer
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(
		starter,
		WithLogger(log.New(&logs, "", 0)),
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
		WithIdleTimeout(time.Millisecond),
	)

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 35}); err != nil {
		t.Fatal(err)
	}
	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 35}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool {
		return manager.ActiveSessions() == 0
	})
	output := logs.String()
	for _, expected := range []string{"event=stream_create", "event=stream_ready", "event=stream_reuse", "event=stream_cleanup"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("logs missing %q:\n%s", expected, output)
		}
	}
}

func TestSessionManagerRestartsFailedChannel(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithRestartDelay(time.Millisecond), WithStopTimeout(time.Second))

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 13}); err != nil {
		t.Fatal(err)
	}
	firstProcess := starter.processes[0]
	firstProcess.fail()

	waitFor(t, time.Second, func() bool {
		return starter.starts == 2 && manager.ActiveSessions() == 1
	})
	if firstProcess.stopped {
		t.Fatal("failed process should not be stopped again")
	}
}

func TestSessionManagerRetriesFailedRestart(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithRestartDelay(time.Millisecond), WithStopTimeout(time.Second))

	if err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 17}); err != nil {
		t.Fatal(err)
	}
	starter.failStartsRemaining = 1
	starter.processes[0].fail()

	waitFor(t, time.Second, func() bool {
		return starter.starts == 2 && manager.ActiveSessions() == 1
	})
}

func TestSessionManagerStopAllStopsProcessesAndPreventsRestart(t *testing.T) {
	starter := &fakeSessionStarter{}
	manager := NewSessionManager(starter, WithRestartDelay(time.Millisecond), WithStopTimeout(time.Second))

	if err := manager.StartChannels(context.Background(), []ChannelRequest{{ChannelID: 21}, {ChannelID: 22}}); err != nil {
		t.Fatal(err)
	}
	manager.StopAll(context.Background())

	if manager.ActiveSessions() != 0 {
		t.Fatalf("active sessions = %d, want 0", manager.ActiveSessions())
	}
	for _, process := range starter.processes {
		if !process.stopped {
			t.Fatalf("process was not stopped: %#v", process)
		}
	}
	time.Sleep(10 * time.Millisecond)
	if starter.starts != 2 {
		t.Fatalf("starts = %d, want 2; stopped channels restarted", starter.starts)
	}
}

func TestSessionManagerReturnsStartErrors(t *testing.T) {
	starter := &fakeSessionStarter{err: errors.New("start failed")}
	manager := NewSessionManager(starter)

	err := manager.EnsureChannel(context.Background(), ChannelRequest{ChannelID: 15})
	if err == nil {
		t.Fatal("err = nil")
	}
	if manager.ActiveSessions() != 0 {
		t.Fatalf("active sessions = %d, want 0", manager.ActiveSessions())
	}
}

func TestNextTransitionDelay(t *testing.T) {
	now := time.Date(2026, 7, 11, 19, 30, 0, 0, time.UTC)
	end := now.Add(30 * time.Minute)
	if delay := nextTransitionDelay(now, end); delay != 30*time.Minute {
		t.Fatalf("delay = %s, want 30m", delay)
	}
	if delay := nextTransitionDelay(now, now.Add(-time.Second)); delay != 0 {
		t.Fatalf("expired delay = %s, want 0", delay)
	}
}

func TestSessionManagerTransitionsToNextProgram(t *testing.T) {
	starter := &fakeSessionStarter{uniquePlaybackURL: true}
	provider := &fakeChannelRequestProvider{}
	manager := NewSessionManager(
		starter,
		WithChannelRequestProvider(provider),
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
	)

	firstEnd := time.Now().UTC().Add(10 * time.Millisecond)
	secondEnd := time.Now().UTC().Add(time.Hour)
	provider.next = ChannelRequest{
		ChannelID: 7,
		ProgramID: 2,
		EndTime:   secondEnd,
		Request: Request{
			Program:      Program{MediaType: MediaTypeMovie, MovieID: 102},
			SeekPosition: 0,
			Presentation: presentation.BroadcastPresentation{
				NowPlaying: &presentation.TextOverlay{Text: "Second Program"},
			},
		},
	}

	err := manager.EnsureChannel(context.Background(), ChannelRequest{
		ChannelID: 7,
		ProgramID: 1,
		EndTime:   firstEnd,
		Request: Request{
			Program:      Program{MediaType: MediaTypeMovie, MovieID: 101},
			SeekPosition: 12,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	firstURL := starter.processes[0].playback.URL
	firstProcess := starter.processes[0]

	waitFor(t, time.Second, func() bool {
		return starter.starts == 2 && manager.ActiveSessions() == 1
	})

	if !firstProcess.stopped {
		t.Fatal("previous process was not stopped during rollover")
	}
	playback, ok := manager.Playback(7)
	if !ok {
		t.Fatal("playback missing after rollover")
	}
	if playback.URL != firstURL {
		t.Fatalf("playback URL = %q, want stable URL %q", playback.URL, firstURL)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if starter.requests[1].ProgramID != 2 || starter.requests[1].Request.Program.MovieID != 102 {
		t.Fatalf("second request = %#v", starter.requests[1])
	}
	if starter.requests[1].Request.Presentation.NowPlaying == nil ||
		starter.requests[1].Request.Presentation.NowPlaying.Text != "Second Program" {
		t.Fatalf("second presentation = %#v", starter.requests[1].Request.Presentation)
	}
}

func TestSessionManagerStopsRolloverWhenChannelIsStopped(t *testing.T) {
	starter := &fakeSessionStarter{}
	provider := &fakeChannelRequestProvider{
		next: ChannelRequest{ChannelID: 9, ProgramID: 2, EndTime: time.Now().UTC().Add(time.Hour)},
	}
	manager := NewSessionManager(
		starter,
		WithChannelRequestProvider(provider),
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
	)

	err := manager.EnsureChannel(context.Background(), ChannelRequest{
		ChannelID: 9,
		ProgramID: 1,
		EndTime:   time.Now().UTC().Add(20 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	manager.StopChannel(context.Background(), 9)

	time.Sleep(40 * time.Millisecond)
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1", starter.starts)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
}

func TestSessionManagerStopsSessionWhenNextProgramIsMissing(t *testing.T) {
	starter := &fakeSessionStarter{}
	provider := &fakeChannelRequestProvider{ok: false}
	manager := NewSessionManager(
		starter,
		WithChannelRequestProvider(provider),
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
	)

	err := manager.EnsureChannel(context.Background(), ChannelRequest{
		ChannelID: 10,
		ProgramID: 1,
		EndTime:   time.Now().UTC().Add(10 * time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	firstProcess := starter.processes[0]

	waitFor(t, time.Second, func() bool {
		return provider.calls == 1 && manager.ActiveSessions() == 0
	})
	if !firstProcess.stopped {
		t.Fatal("process was not stopped when next program was missing")
	}
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1", starter.starts)
	}
}

func TestSessionManagerDoesNotCreateDuplicateRolloverTimers(t *testing.T) {
	starter := &fakeSessionStarter{}
	provider := &fakeChannelRequestProvider{
		next: ChannelRequest{
			ChannelID: 12,
			ProgramID: 2,
			EndTime:   time.Now().UTC().Add(time.Hour),
		},
	}
	manager := NewSessionManager(
		starter,
		WithChannelRequestProvider(provider),
		WithRestartDelay(time.Hour),
		WithStopTimeout(time.Second),
	)

	end := time.Now().UTC().Add(20 * time.Millisecond)
	request := ChannelRequest{ChannelID: 12, ProgramID: 1, EndTime: end}
	if err := manager.EnsureChannel(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := manager.EnsureChannel(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	waitFor(t, time.Second, func() bool {
		return provider.calls == 1 && starter.starts == 2
	})
	time.Sleep(30 * time.Millisecond)
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if starter.starts != 2 {
		t.Fatalf("starts = %d, want 2", starter.starts)
	}
}

type fakeSessionStarter struct {
	starts              int
	err                 error
	failStartsRemaining int
	processes           []*fakeSessionProcess
	requests            []ChannelRequest
	uniquePlaybackURL   bool
	startBlock          chan struct{}
}

func (s *fakeSessionStarter) StartSession(ctx context.Context, request ChannelRequest) (SessionProcess, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.failStartsRemaining > 0 {
		s.failStartsRemaining--
		return nil, errors.New("temporary start failure")
	}
	s.starts++
	if s.startBlock != nil {
		<-s.startBlock
	}
	s.requests = append(s.requests, request)
	playbackURL := "http://retrotv.test/streams/channel/" + strconv.FormatInt(request.ChannelID, 10)
	if s.uniquePlaybackURL {
		playbackURL += "/start/" + strconv.Itoa(s.starts)
	}
	process := &fakeSessionProcess{
		playback: Playback{
			URL:          playbackURL,
			SeekPosition: request.Request.SeekPosition,
		},
		done: make(chan error),
	}
	s.processes = append(s.processes, process)
	return process, nil
}

type fakeSessionProcess struct {
	playback Playback
	done     chan error
	stopped  bool
}

func (p *fakeSessionProcess) Playback() Playback {
	return p.playback
}

func (p *fakeSessionProcess) Stop(ctx context.Context) error {
	if !p.stopped {
		p.stopped = true
		close(p.done)
	}
	return nil
}

func (p *fakeSessionProcess) Done() <-chan error {
	return p.done
}

func (p *fakeSessionProcess) fail() {
	close(p.done)
}

type fakeChannelRequestProvider struct {
	calls int
	next  ChannelRequest
	ok    bool
	err   error
}

func (p *fakeChannelRequestProvider) CurrentChannelRequest(ctx context.Context, channelID int64, now time.Time) (ChannelRequest, bool, error) {
	p.calls++
	if p.err != nil {
		return ChannelRequest{}, false, p.err
	}
	if !p.ok && p.next.ChannelID == 0 {
		return ChannelRequest{}, false, nil
	}
	return p.next, true, nil
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}
