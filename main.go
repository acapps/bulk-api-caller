package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

var concurrency = 5 // How many concurrent web requests should be used?

// Given a fileName, determine the path of the executable and open the file in the current directory.
func openFile(fileName string) (*os.File, error) {

	// Determine the path to where the application is running.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := path.Dir(ex)

	// Open the file in the directory where the application is running and the fileName provided.
	return os.Open(fmt.Sprintf("%s/%s", exPath, fileName))
}

func iterateFile(file *os.File) {

	sem := make(chan bool, concurrency)

	// Read in each line of the file.
	r := bufio.NewReader(file)

	for {
		urlToCall, err := readLine(r) // Reach each line out of the file.
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("An error occurred parsing input string: %s\n", err)
			continue
		}

		sem <- true // block until a worker is ready.

		go func(url string) {
			defer func() { <-sem }()

			openUrl(url)
		}(urlToCall)
	}

	// Let all in-flight requests finish.
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func readLine(r *bufio.Reader) (string, error) {
	var (
		isPrefix       = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

// Opens the requested url and logs the response.
func openUrl(url string) {

	t1 := time.Now()
	resp, err := http.Get(url)
	t2 := time.Now()
	if err != nil {
		log.Println(err)
	}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("An error occurred opening url: %s\n", err))
	}

	diff := t2.Sub(t1) // Time passed.
	log.Printf("'%s', '%dms'\n", response, diff.Nanoseconds()/int64(time.Millisecond))

}

func main() {

	fileInput := flag.String("fileName", "", "File with messages to send, must be in same directory.")
	concurrencyInput := flag.Int("threads", 5, "Number of concurrent requests, defaults to 5.")

	flag.Parse()

	// If fileName is empty then exit.
	if *fileInput == "" {
		panic("fileName must be provided.")
	}

	// Set global state to use the provide apiKey and level of concurrency.
	concurrency = *concurrencyInput

	// Open the file in the directory where the application is running and the fileName provided.
	file, err := openFile(*fileInput)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	t1 := time.Now()
	log.Println("Starting API requests:")

	iterateFile(file)
	t2 := time.Now()
	diff := t2.Sub(t1) // Time passed.

	log.Printf("Finished API requests! Total time elapsed: %d seconds", diff.Nanoseconds()/int64(time.Second))
}
