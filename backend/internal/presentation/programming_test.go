package presentation

import (
	"testing"
	"time"

	"retrotv/internal/models"
)

func TestResolveViewerProgramsHidesAssetsAndCoalescesResumedProgram(t *testing.T) {
	start := time.Date(2026, 7, 16, 20, 0, 0, 0, time.UTC)
	schedule := []models.ScheduledProgram{
		{ID: 1, MediaID: 10, MediaType: "tv", EpisodePath: "family-guy.mkv", StartTime: start, EndTime: start.Add(22 * time.Minute)},
		{ID: 2, MediaID: 20, MediaType: "asset", StartTime: start.Add(22 * time.Minute), EndTime: start.Add(22*time.Minute + 15*time.Second)},
		{ID: 3, MediaID: 21, MediaType: "asset", StartTime: start.Add(22*time.Minute + 15*time.Second), EndTime: start.Add(23 * time.Minute)},
		{ID: 4, MediaID: 10, MediaType: "tv", EpisodePath: "family-guy.mkv", StartTime: start.Add(23 * time.Minute), EndTime: start.Add(30 * time.Minute)},
		{ID: 5, MediaID: 11, MediaType: "tv", EpisodePath: "king-of-the-hill.mkv", StartTime: start.Add(30 * time.Minute), EndTime: start.Add(time.Hour)},
	}

	result := ResolveViewerPrograms(schedule, start.Add(22*time.Minute+30*time.Second))
	if result.Current == nil || result.Current.ID != 1 {
		t.Fatalf("current = %#v, want Family Guy", result.Current)
	}
	if result.Next == nil || result.Next.ID != 5 {
		t.Fatalf("next = %#v, want King of the Hill", result.Next)
	}
	if !result.Current.EndTime.Equal(start.Add(30 * time.Minute)) {
		t.Fatalf("current end = %s, want 20:30", result.Current.EndTime)
	}
}

func TestResolveViewerProgramsReturnsCurrentRealProgram(t *testing.T) {
	start := time.Date(2026, 7, 16, 20, 0, 0, 0, time.UTC)
	schedule := []models.ScheduledProgram{
		{ID: 1, MediaID: 10, MediaType: "movie", StartTime: start, EndTime: start.Add(time.Hour)},
		{ID: 2, MediaID: 11, MediaType: "movie", StartTime: start.Add(time.Hour), EndTime: start.Add(2 * time.Hour)},
	}
	result := ResolveViewerPrograms(schedule, start.Add(10*time.Minute))
	if result.Current == nil || result.Current.ID != 1 || result.Next == nil || result.Next.ID != 2 {
		t.Fatalf("result = %#v", result)
	}
}
