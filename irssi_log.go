/*
 * Parser for Irssi logs
 */

package irssi_log

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

type EntryType int

const (
	LogOpen EntryType = iota
	Join
	ChannelSummary
	Mode
	JoinSync
	Message
	Quit
	NickChange
	DayChange
	LogClosed
	NowTalking
	Emote
	Topic
	Kick
	Part
	YourNickChange
	ServerMode
	ChannelNotice
	IgnoreThis
	ServerNotice
	BansNone
)

type LogEntry struct {
	// Raw line
	Line string

	// Parsed time
	Time time.Time

	// Type of line
	Type EntryType

	// Channel, if available
	Channel string

	// Nick, if available
	Nick string

	// user@host, if available
	UserHost string

	// Text, if applicable. e.g., message text
	Text string
}

const LogOpenTimeLayout = "Mon Jan 02 15:04:05 2006"

// ParseLog reads lines of an Irssi log and generates an ordered slice
// of LogEntrys
func ParseLog(file *os.File, lineLimit int, location *time.Location) (
	[]*LogEntry, error) {
	scanner := bufio.NewScanner(file)

	lineCount := 0

	var entries []*LogEntry

	var currentDate time.Time

	for scanner.Scan() {
		lineCount++

		entry, err := ParseLine(scanner.Text(), location, currentDate)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse line: %s", err.Error())
		}

		if entry.Type == BansNone {
			log.Printf("Parsed line %q", entry)
		}

		entries = append(entries, entry)

		if entry.Type == LogOpen || entry.Type == DayChange {
			currentDate = time.Date(entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), 0, 0, 0, 0, location)
		}

		if lineLimit > 0 && lineCount >= lineLimit {
			return entries, nil
		}
	}

	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("Line scan failure: %s", err.Error())
	}

	return entries, nil
}

// ParseLine parses an Irssi log line
func ParseLine(line string, location *time.Location, currentDate time.Time) (
	*LogEntry, error) {

	// Log open type.

	logOpenPattern := regexp.MustCompile("^--- Log opened (.+)$")

	logOpenMatches := logOpenPattern.FindStringSubmatch(line)
	if logOpenMatches != nil {
		entryTime, err := time.ParseInLocation(LogOpenTimeLayout, logOpenMatches[1],
			location)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse timestamp: %s: %s",
				logOpenMatches[1], err.Error())
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: LogOpen,
		}, nil
	}

	// Join type.

	joinPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) \\[(\\S+?)\\] has joined (\\S+)$")

	joinMatches := joinPattern.FindStringSubmatch(line)
	if joinMatches != nil {
		entryTime, err := clockToTime(joinMatches[1], joinMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:     line,
			Time:     entryTime,
			Type:     Join,
			Channel:  joinMatches[5],
			Nick:     joinMatches[3],
			UserHost: joinMatches[4],
		}, nil
	}

	// Channel summary

	summaryPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- Irssi: (\\S+): Total of \\d+ nicks \\[\\d+ ops, \\d+ halfops, \\d+ voices, \\d+ normal\\]$")

	summaryMatches := summaryPattern.FindStringSubmatch(line)
	if summaryMatches != nil {
		entryTime, err := clockToTime(summaryMatches[1], summaryMatches[2],
			currentDate, location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    ChannelSummary,
			Channel: summaryMatches[3],
		}, nil
	}

	// Mode change

	// TODO: Parse out the modes and who/what targeted

	modePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- mode/(\\S+) \\[.+\\] by (\\S+)$")

	modeMatches := modePattern.FindStringSubmatch(line)
	if modeMatches != nil {
		entryTime, err := clockToTime(modeMatches[1], modeMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    Mode,
			Channel: modeMatches[3],
			Nick:    modeMatches[4],
		}, nil
	}

	// Channel sync

	syncPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- Irssi: Join to (\\S+) was synced in \\d+ secs$")

	syncMatches := syncPattern.FindStringSubmatch(line)
	if syncMatches != nil {
		entryTime, err := clockToTime(syncMatches[1], syncMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    JoinSync,
			Channel: syncMatches[3],
		}, nil
	}

	// Channel message

	// Text can be totally blank
	messagePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) <(.)(\\S+)> (.*)$")

	messageMatches := messagePattern.FindStringSubmatch(line)
	if messageMatches != nil {
		entryTime, err := clockToTime(messageMatches[1], messageMatches[2],
			currentDate, location)
		if err != nil {
			return nil, err
		}

		// TODO: Get channel

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: Message,
			Nick: messageMatches[4],
			Text: messageMatches[5],
		}, nil
	}

	// Quit

	quitPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) \\[(\\S+)\\] has quit \\[(.*)\\]$")

	quitMatches := quitPattern.FindStringSubmatch(line)
	if quitMatches != nil {
		entryTime, err := clockToTime(quitMatches[1], quitMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		// TODO: Get channel

		return &LogEntry{
			Line:     line,
			Time:     entryTime,
			Type:     Quit,
			Nick:     quitMatches[3],
			UserHost: quitMatches[4],
			Text:     quitMatches[5],
		}, nil
	}

	// Nick change

	nickPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) is now known as (\\S+)$")

	nickMatches := nickPattern.FindStringSubmatch(line)
	if nickMatches != nil {
		entryTime, err := clockToTime(nickMatches[1], nickMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: NickChange,
			Nick: nickMatches[3],
			Text: nickMatches[4],
		}, nil
	}

	// Day change

	dayPattern := regexp.MustCompile("^--- Day changed (.+)$")

	dayMatches := dayPattern.FindStringSubmatch(line)
	if dayMatches != nil {
		timeLayout := "Mon Jan 02 2006"
		entryTime, err := time.ParseInLocation(timeLayout, dayMatches[1], location)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse timestamp: %s: %s", dayMatches[1],
				err.Error())
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: DayChange,
		}, nil
	}

	// Log closed

	closePattern := regexp.MustCompile("^--- Log closed (.+)$")

	closeMatches := closePattern.FindStringSubmatch(line)
	if closeMatches != nil {
		timeLayout := "Mon Jan 02 15:04:05 2006"
		entryTime, err := time.ParseInLocation(timeLayout, closeMatches[1],
			location)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse timestamp: %s: %s",
				closeMatches[1], err.Error())
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: LogClosed,
		}, nil
	}

	// Now talking in

	nowPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- Irssi: You are now talking in (\\S+)$")

	nowMatches := nowPattern.FindStringSubmatch(line)
	if nowMatches != nil {
		entryTime, err := clockToTime(nowMatches[1], nowMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    NowTalking,
			Channel: nowMatches[3],
		}, nil
	}

	// Channel emote

	emotePattern := regexp.MustCompile("^(\\d{2}):(\\d{2})  \\* (\\S+) (.*)$")

	emoteMatches := emotePattern.FindStringSubmatch(line)
	if emoteMatches != nil {
		entryTime, err := clockToTime(emoteMatches[1], emoteMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: Emote,
			Nick: emoteMatches[3],
			Text: emoteMatches[4],
		}, nil
	}

	// Topic change

	topicPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) changed the topic of (\\S+) to: (.*)$")

	topicMatches := topicPattern.FindStringSubmatch(line)
	if topicMatches != nil {
		entryTime, err := clockToTime(topicMatches[1], topicMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    Topic,
			Nick:    topicMatches[3],
			Channel: topicMatches[4],
			Text:    topicMatches[5],
		}, nil
	}

	// Kick

	kickPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) was kicked from (\\S+) by (\\S+) \\[(.*)\\]$")

	kickMatches := kickPattern.FindStringSubmatch(line)
	if kickMatches != nil {
		entryTime, err := clockToTime(kickMatches[1], kickMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		// TODO: 2 nicks

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    Kick,
			Nick:    kickMatches[3],
			Channel: kickMatches[4],
			Text:    kickMatches[6],
		}, nil
	}

	// Part

	partPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) \\[(\\S+)\\] has left (\\S+) \\[(.*)\\]$")

	partMatches := partPattern.FindStringSubmatch(line)
	if partMatches != nil {
		entryTime, err := clockToTime(partMatches[1], partMatches[2], currentDate,
			location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:     line,
			Time:     entryTime,
			Type:     Part,
			Nick:     partMatches[3],
			UserHost: partMatches[4],
			Channel:  partMatches[5],
			Text:     partMatches[6],
		}, nil
	}

	// Your nick change

	yourNickPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- You're now known as (\\S+)$")

	yourNickMatches := yourNickPattern.FindStringSubmatch(line)
	if yourNickMatches != nil {
		entryTime, err := clockToTime(yourNickMatches[1], yourNickMatches[2],
			currentDate, location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: YourNickChange,
			Nick: yourNickMatches[3],
		}, nil
	}

	// Server changed mode

	serverModePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- ServerMode/(\\S+) \\[(.+)\\] by (\\S+)$")

	serverModeMatches := serverModePattern.FindStringSubmatch(line)
	if serverModeMatches != nil {
		entryTime, err := clockToTime(serverModeMatches[1], serverModeMatches[2],
			currentDate, location)
		if err != nil {
			return nil, err
		}

		// TODO: Parse modes

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    ServerMode,
			Channel: serverModeMatches[3],
			Text:    serverModeMatches[4],
			Nick:    serverModeMatches[5],
		}, nil
	}

	// Notice to the channel

	channelNoticePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -(\\S+):[+@]?(\\S+)- (.*)$")

	channelNoticeMatches := channelNoticePattern.FindStringSubmatch(line)
	if channelNoticeMatches != nil {
		entryTime, err := clockToTime(channelNoticeMatches[1],
			channelNoticeMatches[2], currentDate, location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    ChannelNotice,
			Nick:    channelNoticeMatches[3],
			Text:    channelNoticeMatches[5],
			Channel: channelNoticeMatches[4],
		}, nil
	}

	// Keepnick plugin line.
	// Just ignore it.

	keepnickPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- Keepnick:")

	if keepnickPattern.FindStringSubmatch(line) != nil {
		return &LogEntry{Type: IgnoreThis}, nil
	}

	// Server notice

	serverNoticePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) !(\\S+) (.*)$")

	serverNoticeMatches := serverNoticePattern.FindStringSubmatch(line)
	if serverNoticeMatches != nil {
		entryTime, err := clockToTime(serverNoticeMatches[1],
			serverNoticeMatches[2], currentDate, location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line: line,
			Time: entryTime,
			Type: ServerNotice,
			Nick: serverNoticeMatches[3],
			Text: serverNoticeMatches[4],
		}, nil
	}

	// Ban check: None

	bansNonePattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- Irssi: No bans in channel (\\S+)$")

	bansNoneMatches := bansNonePattern.FindStringSubmatch(line)
	if bansNoneMatches != nil {
		entryTime, err := clockToTime(bansNoneMatches[1], bansNoneMatches[2],
			currentDate, location)
		if err != nil {
			return nil, err
		}

		return &LogEntry{
			Line:    line,
			Time:    entryTime,
			Type:    BansNone,
			Channel: bansNoneMatches[3],
		}, nil
	}

	return nil, fmt.Errorf("Unrecognized line: %s", line)
}

// clockToTime takes a timestamp like HH:MM and makes a time.Time type.
// It places the HH:MM in a given date by using currentDate
func clockToTime(hour string, minutes string, currentDate time.Time, location *time.Location) (time.Time, error) {
	h, err := strconv.Atoi(hour)
	if err != nil {
		return time.Time{}, fmt.Errorf("Unable to parse hour from timestamp: %s: %s", hour, err.Error())
	}

	m, err := strconv.Atoi(minutes)
	if err != nil {
		return time.Time{}, fmt.Errorf("Unable to parse minute from timestamp: %s: %s", minutes[1], err.Error())
	}

	entryTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), h, m, 0, 0, location)

	return entryTime, nil
}
