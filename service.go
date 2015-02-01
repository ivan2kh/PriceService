package main

import (
	//	"encoding/json"
	//	"bufio"
	//	"encoding/xml"
	//	"fmt"
	"log"
	"net/http"
	//"os"

	//"io/ioutil"
	//"io"
	//	"os/exec"
	//"time"
)

var docParser *DocumentParser

func PriceList(rw http.ResponseWriter, req *http.Request) {
	if req.Body == nil || req.Method == "GET" {
		rw.WriteHeader(500)
		rw.Write([]byte("Use POST request with text in body"))
		return
	}
	log.Println("New request")

	rw.Header().Set("Content-Type", "text/plain")
	defer req.Body.Close()
	docParser.ParseFacts(req.Body, rw)
	// if err := cmd.Wait(); err != nil {
	// 	log.Panicln(err)
	// }
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
	log.Fatal(http.ListenAndServe(":55924", nil))
}
