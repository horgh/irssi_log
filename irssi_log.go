/*
 * Parser for Irssi logs
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
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
	DayChange
)

type LogEntry struct {
	Time      time.Time
	EntryType EntryType
	Channel   string
	Nick      string
	UserHost  string
	Text      string
}

func main() {
	logFile := flag.String("log-file", "", "Path to a log file to read.")
	lineLimit := flag.Int("line-limit", 0, "Limit number of lines to read. 0 for entire log.")
	locationString := flag.String("location", "America/Vancouver", "Time zone location.")

	flag.Parse()

	if len(*logFile) == 0 {
		log.Print("You must specify a log file.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *lineLimit < 0 {
		log.Print("You must specify a line limit >= 0.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*locationString) == 0 {
		log.Print("You must specify a location.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	location, err := time.LoadLocation(*locationString)
	if err != nil {
		log.Printf("Invalid location: %s", err.Error())
		os.Exit(1)
	}

	fh, err := os.Open(*logFile)
	if err != nil {
		log.Printf("Unable to open file: %s: %s", *logFile, err.Error())
		os.Exit(1)
	}
	defer fh.Close()

	entries, err := parseLog(fh, *lineLimit, location)
	if err != nil {
		log.Printf("Unable to parse log: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Parsed %d entries.", len(entries))

	log.Print("Done!")
}

// parseLog reads lines of an Irssi log and generates an ordered slice
// of LogEntrys
func parseLog(file *os.File, lineLimit int, location *time.Location) (
	[]*LogEntry, error) {
	scanner := bufio.NewScanner(file)

	lineCount := 0

	var entries []*LogEntry

	var currentDate time.Time

	for scanner.Scan() {
		lineCount++

		entry, err := parseLine(scanner.Text(), location, currentDate)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse line: %s", err.Error())
		}

		log.Printf("Parsed line %q", entry)

		if entry.EntryType == LogOpen || entry.EntryType == DayChange {
			currentDate = time.Date(entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), 0, 0, 0, 0, location)
		}

		if lineCount >= lineLimit {
			return entries, nil
		}
	}

	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("Line scan failure: %s", err.Error())
	}

	return entries, nil
}

// parseLine parses an Irssi log line
func parseLine(line string, location *time.Location, currentDate time.Time) (*LogEntry, error) {
	// Log open type.

	// Format of the log open time
	timeLayout := "Mon Jan 02 15:04:05 2006"

	logOpenPattern := regexp.MustCompile("^--- Log opened (.+)$")

	matches := logOpenPattern.FindStringSubmatch(line)
	if matches != nil {
		entryTime, err := time.ParseInLocation(timeLayout, matches[1], location)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse timestamp: %s: %s", matches[1],
				err.Error())
		}

		return &LogEntry{
			Time:      entryTime,
			EntryType: LogOpen,
		}, nil
	}

	// Join type.

	joinPattern := regexp.MustCompile("^(\\d{2}):(\\d{2}) -!- (\\S+) \\[(\\S+?)\\] has joined (\\S+)$")
	joinMatches := joinPattern.FindStringSubmatch(line)
	if joinMatches != nil {
		// TODO: Convert hour/minute into ints
		entryTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), hour, minute, 0, 0, location)

		return &LogEntry{
			Time:      entryTime,
			EntryType: Join,
			Channel:   matches[5],
			Nick:      matches[3],
			UserHost:  matches[4],
		}, nil
	}

	return nil, fmt.Errorf("Unrecognized line: %s", line)
}
