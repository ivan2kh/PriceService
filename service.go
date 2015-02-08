package main

import (
	//	"encoding/json"
	//	"bufio"
	//	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"

	"io"
	"io/ioutil"
	//	"os/exec"
	//"time"
	"compress/gzip"
)

var docParser *DocumentParser

func PriceList(rw http.ResponseWriter, req *http.Request) {
	const MaxFileSize = 100 * 1024 * 1024
	if req.ContentLength > MaxFileSize {
		msg := fmt.Sprintf("request too large: %v > %v", req.ContentLength, MaxFileSize)
		http.Error(rw, msg, http.StatusExpectationFailed)
		return
	}

	if req.Body == nil || req.Method == "GET" {
		rw.WriteHeader(500)
		rw.Write([]byte("Use POST request with text in body"))
		return
	}
	log.Println("New request")

	rw.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()
	docParser.ParseFacts(req.Body, rw, MaxFileSize)
}

func PriceFile(rw http.ResponseWriter, req *http.Request) {
	const MaxFileSize = 100 * 1024 * 1024
	if req.ContentLength > MaxFileSize {
		msg := fmt.Sprintf("request too large: %v > %v", req.ContentLength, MaxFileSize)
		http.Error(rw, msg, http.StatusExpectationFailed)
		return
	}

	if req.Body == nil || req.Method == "GET" {
		rw.WriteHeader(500)
		rw.Write([]byte("Use POST request with text in body"))
		return
	}
	defer req.Body.Close()

	temp, err := ioutil.TempFile("", "drugs")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Created temp file %v", temp.Name())

	defer func() {
		err = os.Remove(temp.Name())
		if err != nil {
			log.Fatalf("Remove temp err %v", err)
		}
	}()

	defer func() {
		err = temp.Close()
		if err != nil {
			log.Fatalf("Close temp err %v", err)
		}
	}()

	_, err = io.Copy(temp, req.Body)
	req.Body.Close()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("New PriceFile request")
	rw.Header().Set("Content-Type", "text/plain")

	temp.Seek(0, 0)
	gzipReader, err := gzip.NewReader(temp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	docParser.ParseFacts(gzipReader, rw, MaxFileSize*5)
}

//1. use zipped data transfer
//2. run one parser instance at startup
//3. use multithreading if necessary
//4. wait till all the data goes from client. if number of items is wrong, alert
func main() {
	newDocParser, err := CreateDocumentParser()
	if err != nil {
		log.Panicln(err)
	}
	docParser = newDocParser
	http.HandleFunc("/pricelist", makeGzipHandler(PriceList))
	http.HandleFunc("/pricelist_file", makeGzipWriterHandler(PriceFile))
	log.Fatal(http.ListenAndServe(":55924", nil))
}
