/*
 * Markov text generation using an Irssi channel log
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/horgh/irssi_log/suffixarray"
)

func main() {
	file := flag.String("file", "", "Path to a file to read. Should be one block of text. You can process a log to get this using the messages_to_string program.")
	sentenceLength := flag.Int("sentence-length", 12, "Numbers of words to generate.")
	k := flag.Int("k", 2, "How many preceding words to take into account when picking the next.")

	flag.Parse()

	if len(*file) == 0 {
		log.Print("You must specify a file.")
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

	logMemory()

	log.Printf("Reading file...")
	text, err := readFile(*file)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	logMemory()

	log.Printf("Generating suffix array...")
	a, err := suffixarray.Build(text)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	logMemory()

	log.Printf("Sorting suffix array...")
	s, err := suffixarray.Sort(a)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	logMemory()

	log.Printf("Generating text...")
	rand.Seed(time.Now().UnixNano())
	sentence, err := generateTextFromSuffixArray(s, *sentenceLength, *k)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
	log.Printf("Generated: %s", sentence)
}

// readFile reads in a file as a string
func readFile(filename string) (string, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("Unable to open file: %s: %s", filename, err.Error())
	}
	defer fh.Close()

	reader := bufio.NewReader(fh)
	text := ""
	buf := make([]byte, 10*1024*1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("Read error: %s", err.Error())
		}
		text += string(buf[0:n])
	}

	return text, nil
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

// logMemory logs the memory used.
func logMemory() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	var allocMiB float64 = float64(mem.Alloc) / 1024.0 / 1024.0
	var sysMiB float64 = float64(mem.Sys) / 1024.0 / 1024.0
	log.Printf("Alloc: %.2f MiB Sys: %.2f MiB", allocMiB, sysMiB)
}
