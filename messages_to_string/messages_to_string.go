/*
 * This program takes an Irssi channel log and extracts the message text.
 *
 * It appends them all together and writes them to a file.
 *
 * This is to make generating random text from the messages quicker.
 */

package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"regexp"
	"strings"
	"summercat.com/irssi_log"
	"time"
)

var urlPattern = regexp.MustCompile("https?:")

func main() {
	logFile := flag.String("log-file", "", "Path to a log file to read.")
	outFile := flag.String("out-file", "", "Path to file to write.")
	lineLimit := flag.Int("line-limit", 0, "Limit number of lines to read. 0 for entire log.")
	locationString := flag.String("location", "America/Vancouver", "Time zone location.")

	flag.Parse()

	if len(*logFile) == 0 {
		log.Print("You must specify a log file.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*outFile) == 0 {
		log.Print("You must specify an output file.")
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

	log.Printf("Parsing log...")
	entries, err := irssi_log.ParseLog(fh, *lineLimit, location)
	if err != nil {
		log.Printf("Unable to parse log: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Writing file...")
	ofh, err := os.Create(*outFile)
	if err != nil {
		log.Printf("Unable to open output file: %s: %s", *outFile, err.Error())
		os.Exit(1)
	}
	defer ofh.Close()

	err = writeMessages(ofh, entries)
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}

	log.Printf("Done!")
}

// writeMessages takes the message text and writes them all out to a file.
func writeMessages(fh *os.File, entries []*irssi_log.LogEntry) error {
	writer := bufio.NewWriter(fh)
	defer writer.Flush()

	first := true

	for _, entry := range entries {
		if entry.Type != irssi_log.Message {
			continue
		}

		if strings.HasPrefix(entry.Text, " ") {
			continue
		}

		words := strings.Split(entry.Text, " ")

		for _, word := range words {
			wordTrim := strings.TrimSpace(word)
			if len(wordTrim) == 0 {
				continue
			}

			if urlPattern.MatchString(word) {
				continue
			}

			if !first {
				_, err := writer.WriteString(" ")
				if err != nil {
					return err
				}
			}

			_, err := writer.WriteString(word)
			if err != nil {
				return err
			}

			first = false
		}
	}

	return nil
}
