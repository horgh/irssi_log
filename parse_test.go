package irssi_log

import (
	"errors"
	"testing"
	"time"
)

func TestParseLine(t *testing.T) {
	location, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		t.Fatalf("Invalid location: %s", err.Error())
	}

	currentDate, err := time.ParseInLocation(time.RFC1123,
		"Sun, 27 Mar 2016 15:04:05 PDT", location)
	if err != nil {
		t.Fatalf("Could not parse date: %s", err.Error())
	}

	currentDateZeroSecs := currentDate.Truncate(time.Minute)

	type TestCase struct {
		Line  string
		Entry LogEntry
		Error error
	}

	cases := []TestCase{
		TestCase{
			Line:  "test",
			Entry: LogEntry{},
			Error: errors.New("Invalid line"),
		},
		TestCase{
			Line: "--- Log opened Sun Mar 27 15:04:05 2016",
			Entry: LogEntry{
				Time: currentDate,
				Type: LogOpen,
			},
			Error: nil,
		},
		TestCase{
			Line: "15:04 -!- nick [user@host] has joined #channel",
			Entry: LogEntry{
				Time:     currentDateZeroSecs,
				Type:     Join,
				Nick:     "nick",
				UserHost: "user@host",
				Channel:  "#channel",
			},
			Error: nil,
		},
		TestCase{
			Line: "15:04 -!- Irssi: #channel: Total of 5 nicks [4 ops, 0 halfops, 0 voices, 1 normal]",
			Entry: LogEntry{
				Time:    currentDateZeroSecs,
				Type:    ChannelSummary,
				Channel: "#channel",
			},
			Error: nil,
		},
		TestCase{
			Line: "15:04 -!- mode/#channel [+o nick1] by nick2",
			Entry: LogEntry{
				Time:    currentDateZeroSecs,
				Type:    Mode,
				Channel: "#channel",
				Nick:    "nick2",
			},
			Error: nil,
		},

		// Channel sync
		// Channel message
		// Quit
		// Nick change
		// Day change
		// Log closed
		// Now talking in
		// Channel emote
		// Topic change
		// Kick
		// Part
		// Your nick change
		// Server changed mode
		// Channel notice
		// Keepnick
		// Server notice
		// Ban check none
	}

	for _, testCase := range cases {
		entry, err := ParseLine(testCase.Line, location, currentDate)
		if err != nil {
			if testCase.Error != nil {
				continue
			}
			t.Errorf("Test case with line [%s] failed: %s", testCase.Line, err.Error())
			continue
		}

		if !entryMatches(t, entry, testCase.Entry) {
			continue
		}
	}
}

// entryMatches compares two log entries.
//
// It triggers a test fail if no match.
func entryMatches(t *testing.T, found *LogEntry, wanted LogEntry) bool {
	if found.Type != wanted.Type {
		t.Errorf("Type does not match: Line: %s Found: %d Wanted %d", found.Type,
			wanted.Type)
		return false
	}

	zeroTime := time.Time{}

	if wanted.Time != zeroTime && found.Time != wanted.Time {
		t.Errorf("Time mismatch: Line: %s Wanted %s, have %s", found.Line,
			wanted.Time, found.Time)
		return false
	}

	if wanted.Nick != found.Nick {
		t.Errorf("Nick mismatch: Line: %s Wanted %s, have %s", found.Line,
			wanted.Nick, found.Nick)
		return false
	}

	if wanted.UserHost != found.UserHost {
		t.Errorf("UserHost mismatch: Line: %s Wanted %s, have %s", found.Line,
			wanted.UserHost, found.UserHost)
		return false
	}

	if wanted.Channel != found.Channel {
		t.Errorf("Channel mismatch: Line: %s Wanted %s, have %s", found.Line,
			wanted.Channel, found.Channel)
		return false
	}

	return true
}
