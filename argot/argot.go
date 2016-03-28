/*
 * Markov text generation using an Irssi channel log
 */

package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"summercat.com/irssi_log"
	"time"
)

func main() {
	logFile := flag.String("log-file", "", "Path to a log file to read.")
	lineLimit := flag.Int("line-limit", 0, "Limit number of lines to read. 0 for entire log.")
	locationString := flag.String("location", "America/Vancouver", "Time zone location.")
	sentenceLength := flag.Int("sentence-length", 12, "Numbers of words to generate.")

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

	if *sentenceLength <= 0 {
		log.Print("You must specify a sentence length >= 0.")
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

	entries, err := irssi_log.ParseLog(fh, *lineLimit, location)
	if err != nil {
		log.Printf("Unable to parse log: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Parsed %d log entries.", len(entries))

	words, err := analyzeEntries(entries)
	if err != nil {
		log.Printf("Unable to analyze log entries: %s", err.Error())
		os.Exit(1)
	}

	sentence, err := generateText(words, *sentenceLength)
	if err != nil {
		log.Printf("Unable to generate text: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Sentence: %s", sentence)

	log.Print("Done!")
}

// analyzeEntries takes irssi log entries and determines how often words
// follow another.
//
// The map looks like:
// word -> word -> int
func analyzeEntries(entries []*irssi_log.LogEntry) (map[string]map[string]int,
	error) {
	var words map[string]map[string]int
	words = make(map[string]map[string]int)

	currentWord := ""

	for _, entry := range entries {
		// We only care about messages right now.
		if entry.Type != irssi_log.Message {
			continue
		}

		// Ignore certain lines. Automated text etc.

		if strings.HasPrefix(entry.Text, " ") {
			continue
		}

		re := regexp.MustCompile("[^a-zA-Z0-9,.\\-!(@=_`%+/*#):?' ]")
		if re.MatchString(entry.Text) {
			log.Printf("Ignore line %s", entry.Text)
			continue
		}

		// Break the text into words.
		entryWords := strings.Split(entry.Text, " ")

		for _, word := range entryWords {
			wordTrim := strings.TrimSpace(word)
			if len(wordTrim) == 0 {
				continue
			}

			// Record that this word follows the last.
			if len(currentWord) > 0 {
				_, ok := words[currentWord]
				if !ok {
					words[currentWord] = make(map[string]int)
				}

				words[currentWord][wordTrim]++
			}

			// Set this word as our active one.
			currentWord = wordTrim
		}
	}

	//log.Printf("words %q", words)

	return words, nil
}

// generateText generates some random text given a word frequency mapping.
func generateText(wordCounts map[string]map[string]int, length int) (string, error) {
	// Pick a random word to start at.

	var words []string
	for word, _ := range wordCounts {
		words = append(words, word)
	}

	currentWord := words[rand.Intn(len(words))]

	log.Printf("Start word: %s", currentWord)

	sentence := currentWord
	wordCount := 1

	for ; wordCount < length; wordCount++ {

		// Determine highest frequency words following this word.

		// If there are none recorded following it, pick one at random.

		_, ok := wordCounts[currentWord]
		if !ok {
			currentWord = words[rand.Intn(len(words))]
		} else {
			frequency := 0
			var possibleWords []string

			for word, count := range wordCounts[currentWord] {
				if count < frequency {
					continue
				}

				if count > frequency {
					frequency = 0
					possibleWords = nil
				}

				possibleWords = append(possibleWords, word)
			}

			// Pick one
			currentWord = possibleWords[rand.Intn(len(possibleWords))]
		}

		sentence += " " + currentWord
	}

	return sentence, nil
}
