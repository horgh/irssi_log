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
	"sort"
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
	k := flag.Int("k", 2, "How many preceding words to take into account when picking the next.")

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

	if *k <= 0 {
		log.Print("You must specify k > 0.")
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

	log.Printf("Convert log entries to string...")
	logText, err := messagesToText(entries)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	log.Printf("Generating suffix array...")
	a, err := buildSuffixArray(logText)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	log.Printf("Sorting suffix array...")
	s, err := sortSuffixArray(a)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	log.Printf("Generating text...")
	rand.Seed(time.Now().UnixNano())
	text, err := generateTextFromSuffixArray(s, *sentenceLength, *k)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
	log.Printf("Generated: %s", text)
}

// messagesToText takes log entries and builds one large string of text
// from all of them.
func messagesToText(entries []*irssi_log.LogEntry) (string, error) {
	text := ""

	for _, entry := range entries {
		if entry.Type != irssi_log.Message {
			continue
		}

		// Ignore certain lines. Automated text etc.

		if strings.HasPrefix(entry.Text, " ") {
			continue
		}

		text += entry.Text + " "
	}

	return strings.TrimSpace(text), nil
}

// buildSuffixArray takes a text and generates a suffix array.
//
// Note there is actually an index/suffixarray package in the core library.
func buildSuffixArray(text string) ([]string, error) {
	var suffixes []string
	suffixes = append(suffixes, text[0:])

	for i, c := range text {
		if c == ' ' {
			suffixes = append(suffixes, text[i+1:])
			continue
		}
	}

	return suffixes, nil
}

// sortSuffixArray sorts the suffix array.
func sortSuffixArray(suffixArray []string) ([]string, error) {
	sort.Strings(suffixArray)
	return suffixArray, nil
}

// generateTextFromSuffixArray generates random text.
func generateTextFromSuffixArray(suffixArray []string, length int, k int) (
	string, error) {
	text := ""
	phrase := ""

	for i := 0; i < length; i++ {
		if phrase == "" {
			phrase = getRandomPhrase(suffixArray, k)
			text += phrase + " "
			continue
		}

		// The function must return true for elements past which they are equal
		searchPhrase := phrase + " "
		phraseIndex := sort.Search(
			len(suffixArray),
			func(i int) bool {
				for j := 0; j < len(searchPhrase) && j < len(suffixArray[i]); j++ {
					if searchPhrase[j] == suffixArray[i][j] {
						continue
					}
					if suffixArray[i][j] > searchPhrase[j] {
						return true
					}
					return false
				}

				return true
			})

		// Not found.
		if phraseIndex == len(suffixArray) {
			log.Printf("Phrase %s not found. Picking at random...", phrase)
			phrase = getRandomPhrase(suffixArray, k)
			log.Printf("Chose %s", phrase)
			text += phrase + " "
			continue
		}

		phraseFull := ""
		// Start at zero so our random choice method works.
		// We want to always pick something! (%1==0)
		for j := 0; j+phraseIndex < len(suffixArray) &&
			strings.HasPrefix(suffixArray[j+phraseIndex], phrase+" "); j++ {
			if rand.Int()%(j+1) == 0 {
				phraseFull = suffixArray[j+phraseIndex]
			}
		}

		phrase = getKWords(phraseFull, k, k)
		text += phrase + " "
	}

	return text, nil
}

// getRandomPhrase takes k words from a random selection in the suffix array
func getRandomPhrase(suffixArray []string, k int) string {
	prefix := suffixArray[rand.Intn(len(suffixArray))]
	return getKWords(prefix, 0, k)
}

// getKWords extracts k words from the given string
func getKWords(text string, skip int, count int) string {
	words := ""
	word := ""
	wordCount := 0
	skippedCount := 0

	for _, c := range text {
		if c != ' ' {
			word += string(c)
			continue
		}

		if len(word) == 0 {
			continue
		}

		if skip > 0 && skippedCount < skip {
			skippedCount++
			word = ""
			continue
		}

		if len(words) > 0 {
			words += " "
		}
		words += word
		word = ""

		wordCount++

		if wordCount >= count {
			return words
		}
	}

	return ""
}
