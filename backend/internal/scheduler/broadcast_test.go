package scheduler

import (
	"testing"
	"time"

	"retrotv/internal/models"
)

func TestCurrentBroadcastReturnsCurrentProgramSeekAndNext(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 10, 0, 0, time.UTC)
	current := models.ScheduledProgram{
		ID:        2,
		MediaID:   20,
		StartTime: now.Add(-10 * time.Minute),
		EndTime:   now.Add(20 * time.Minute),
	}
	next := models.ScheduledProgram{
		ID:        3,
		MediaID:   30,
		StartTime: now.Add(20 * time.Minute),
		EndTime:   now.Add(50 * time.Minute),
	}

	broadcast, ok := CurrentBroadcast([]models.ScheduledProgram{
		{
			ID:        1,
			MediaID:   10,
			StartTime: now.Add(-40 * time.Minute),
			EndTime:   now.Add(-10 * time.Minute),
		},
		current,
		next,
	}, now)

	if !ok {
		t.Fatal("ok = false")
	}
	if broadcast.Program.ID != current.ID {
		t.Fatalf("program ID = %d, want %d", broadcast.Program.ID, current.ID)
	}
	if broadcast.SeekPosition != 600 {
		t.Fatalf("seek position = %d, want 600", broadcast.SeekPosition)
	}
	if broadcast.NextProgram == nil || broadcast.NextProgram.ID != next.ID {
		t.Fatalf("next program = %#v", broadcast.NextProgram)
	}
}

func TestCurrentBroadcastReturnsFalseWithoutCurrentProgram(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 10, 0, 0, time.UTC)

	_, ok := CurrentBroadcast([]models.ScheduledProgram{
		{
			ID:        1,
			StartTime: now.Add(10 * time.Minute),
			EndTime:   now.Add(40 * time.Minute),
		},
	}, now)

	if ok {
		t.Fatal("ok = true")
	}
}
