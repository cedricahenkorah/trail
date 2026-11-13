package core

import (
	"fmt"
	"strings"
	"time"
)

// DueStatus is how an item's due date compares with today.
type DueStatus int

const (
	NoDue DueStatus = iota
	Overdue
	DueToday
	Upcoming
)

const duePrefix = " @due("

// WithDue appends an @due(date) marker to text, or returns text if date is empty.
func WithDue(text, date string) string {
	if date == "" {
		return text
	}

	return fmt.Sprintf("%s @due(%s)", text, date)
}

// ParseDue splits a trailing @due(YYYY-MM-DD) off text. ok is false if there isn't one.
func ParseDue(text string) (rest, date string, ok bool) {
	start := strings.LastIndex(text, duePrefix)

	if start == -1 || !strings.HasSuffix(text, ")") {
		return text, "", false
	}

	date = text[start+len(duePrefix) : len(text)-1]

	if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
		return text, "", false
	}

	return text[:start], date, true
}

// DueStatus compares the item's due date with today, both YYYY-MM-DD.
func (i Item) DueStatus(today string) DueStatus {
	switch {
	case i.Due == "":
		return NoDue
	case i.Due < today:
		return Overdue
	case i.Due == today:
		return DueToday
	default:
		return Upcoming
	}
}
