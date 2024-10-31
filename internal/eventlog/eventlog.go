package eventlog

import (
	"fmt"
	"time"

	"github.com/artefactual-sdps/preprocessing-base/internal/enums"
)

type Event struct {
	Name        string
	Message     string
	Outcome     enums.EventOutcome
	StartedAt   time.Time
	CompletedAt time.Time
}

func NewEvent(t time.Time, name string) *Event {
	return &Event{
		Name:      name,
		Outcome:   enums.EventOutcomeUnspecified,
		StartedAt: t,
	}
}

func (e *Event) Complete(t time.Time, outcome enums.EventOutcome, msg string, a ...any) *Event {
	e.CompletedAt = t
	e.Outcome = outcome
	e.Message = fmt.Sprintf(msg, a...)

	return e
}

func (e *Event) Succeed(t time.Time, msg string, a ...any) *Event {
	return e.Complete(t, enums.EventOutcomeSuccess, msg, a...)
}

func (e *Event) IsSuccess() bool {
	return e.Outcome == enums.EventOutcomeSuccess
}
