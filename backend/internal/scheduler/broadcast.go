package scheduler

import (
	"time"

	"retrotv/internal/models"
)

type Broadcast struct {
	Program      models.ScheduledProgram
	NextProgram  *models.ScheduledProgram
	SeekPosition int
}

func CurrentBroadcast(programs []models.ScheduledProgram, now time.Time) (Broadcast, bool) {
	for index, program := range programs {
		if program.StartTime.After(now) || !program.EndTime.After(now) {
			continue
		}
		seekPosition := int(now.Sub(program.StartTime).Seconds())
		if seekPosition < 0 {
			seekPosition = 0
		}
		broadcast := Broadcast{
			Program:      program,
			SeekPosition: seekPosition,
		}
		if index+1 < len(programs) {
			nextProgram := programs[index+1]
			broadcast.NextProgram = &nextProgram
		}
		return broadcast, true
	}
	return Broadcast{}, false
}
