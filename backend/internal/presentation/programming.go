package presentation

import (
	"time"

	"retrotv/internal/models"
)

// ViewerPrograms contains schedule metadata suitable for display to viewers.
// Assets remain in the input schedule but are never exposed as programs.
type ViewerPrograms struct {
	Current *models.ScheduledProgram
	Next    *models.ScheduledProgram
}

func ResolveViewerPrograms(schedule []models.ScheduledProgram, now time.Time) ViewerPrograms {
	actualIndex := -1
	for index, program := range schedule {
		if !program.StartTime.After(now) && program.EndTime.After(now) {
			actualIndex = index
			break
		}
	}
	if actualIndex < 0 {
		return ViewerPrograms{Next: nextRealProgram(schedule, 0, nil)}
	}

	currentIndex := actualIndex
	for currentIndex >= 0 && !viewerProgram(schedule[currentIndex]) {
		currentIndex--
	}
	if currentIndex < 0 {
		return ViewerPrograms{Next: nextRealProgram(schedule, actualIndex+1, nil)}
	}

	current := schedule[currentIndex]
	next := nextRealProgram(schedule, actualIndex+1, &current)
	if next != nil && next.StartTime.After(current.EndTime) {
		// Present interstitial time as part of the surrounding program block,
		// matching the guide while playback continues to use real asset rows.
		current.EndTime = next.StartTime
	}
	return ViewerPrograms{Current: &current, Next: next}
}

func nextRealProgram(schedule []models.ScheduledProgram, start int, current *models.ScheduledProgram) *models.ScheduledProgram {
	for index := start; index < len(schedule); index++ {
		candidate := schedule[index]
		if !viewerProgram(candidate) {
			continue
		}
		if current != nil && sameViewerProgram(*current, candidate) {
			continue
		}
		return &candidate
	}
	return nil
}

func viewerProgram(program models.ScheduledProgram) bool {
	return program.MediaType == "movie" || program.MediaType == "tv"
}

func sameViewerProgram(left models.ScheduledProgram, right models.ScheduledProgram) bool {
	return left.MediaType == right.MediaType && left.MediaID == right.MediaID && left.EpisodePath == right.EpisodePath
}
