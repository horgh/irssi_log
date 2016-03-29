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

var messageIgnorePattern = regexp.MustCompile("[^a-zA-Z0-9,.\\-!(@=_`%+/*#):?' $]")
var urlPattern = regexp.MustCompile("https?:")

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

	log.Printf("Parsing log...")
	entries, err := irssi_log.ParseLog(fh, *lineLimit, location)
	if err != nil {
		log.Printf("Unable to parse log: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Analyzing log entries...")
	words, err := analyzeEntries(entries)
	if err != nil {
		log.Printf("Unable to analyze log entries: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("Generating text...")

	for i := 0; i < 10; i++ {
		sentence, err := generateText(words, *sentenceLength)
		if err != nil {
			log.Printf("Unable to generate text: %s", err.Error())
			os.Exit(1)
		}

		log.Printf("Sentence: %s", sentence)
	}
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

	word0 := ""
	word1 := ""

	for _, entry := range entries {
		// We only care about messages right now.
		if entry.Type != irssi_log.Message {
			continue
		}

		// Ignore certain lines. Automated text etc.

		if strings.HasPrefix(entry.Text, " ") {
			continue
		}

		//if messageIgnorePattern.MatchString(entry.Text) {
		//	log.Printf("Ignore line %s", entry.Text)
		//	continue
		//}

		// Break the text into "words".
		entryWords := strings.Split(entry.Text, " ")

		for _, word := range entryWords {
			wordTrim := strings.TrimSpace(word)
			if len(wordTrim) == 0 {
				continue
			}

			if urlPattern.MatchString(wordTrim) {
				continue
			}

			// Record that this word follows.
			if len(word0) > 0 && len(word1) > 0 {
				phrase := word0 + " " + word1
				_, ok := words[phrase]
				if !ok {
					words[phrase] = make(map[string]int)
				}

				words[phrase][wordTrim]++
			}

			// Update previous word.
			word0 = word1
			word1 = wordTrim
		}
	}

	return words, nil
}

// generateText generates some random text given a word frequency mapping.
func generateText(wordCounts map[string]map[string]int, length int) (string,
	error) {
	// Pick a random word/phrase to start at.

	// Pull out possible phrases.
	var phrases []string
	for phrase, _ := range wordCounts {
		phrases = append(phrases, phrase)
	}

	randomPhrase := phrases[rand.Intn(len(phrases))]
	split := strings.Split(randomPhrase, " ")
	word0 := split[0]
	word1 := split[1]

	log.Printf("Start words: %s %s", word0, word1)

	// Generate text

	sentence := randomPhrase

	for wordCount := 2; wordCount < length; wordCount++ {
		word, wordn0, wordn1 := pickWord(wordCounts, phrases, word0, word1)
		sentence += word
		word0 = wordn0
		word1 = wordn1
	}

	return sentence, nil
}

// pickWord decides on the next word.
//
// We decide this based on the last words.
func pickWord(wordCounts map[string]map[string]int, phrases []string,
	word0 string, word1 string) (string, string, string) {
	phrase := word0 + " " + word1

	// If there are no words recorded following it, pick one at random.

	_, ok := wordCounts[phrase]
	if !ok {
		log.Printf("Nothing known to follow [%s] [%s]", word0, word1)
		phrase = phrases[rand.Intn(len(phrases))]
		split := strings.Split(phrase, " ")
		return ". ", split[0], split[1]
	}

	// Find words that follow the phrase.

	frequency := 0
	var possibleWords []string

	for word, count := range wordCounts[phrase] {
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
	randomWord := possibleWords[rand.Intn(len(possibleWords))]
	return " " + randomWord, word1, randomWord
}
