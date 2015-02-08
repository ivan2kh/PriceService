package main

import (
	"encoding/xml"
	//	"fmt"
	//	"io/ioutil"
	//	"log"
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	//"time"
	"bytes"
	//	"strings"
	"sync"
)

type Document struct {
	DocumentIndex int64 `xml:"di,attr"`
	Name          Val   `xml:"facts>Drug>Name"`
	Form          []Val `xml:"facts>Drug>Form"`
	Dosage        []Val `xml:"facts>Drug>Dosage"`
	TotalQuantity []Val `xml:"facts>Drug>TotalQuantity"`
	EOF           []Val `xml:"facts>EOF>EOF"`
}

type Val struct {
	Value string `xml:"val,attr"`
}

type DocumentParser struct {
	Cmd            *exec.Cmd
	Stdout         io.ReadCloser
	Stderr         io.ReadCloser
	bodyReader     *ReadersQueue
	lastError      string
	documentOffset int64
	sync.Mutex
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func (this *DocumentParser) ParseStderr() {
	scanner := bufio.NewScanner(this.Stderr)
	scanner.Split(ScanLines)
	for {
		for scanner.Scan() {
			this.lastError = scanner.Text()
			//log.Printf("lasterr %v\n", this.lastError)
			if err := scanner.Err(); err != nil {
				log.Printf("reading standard input:", err)
			}
		}
		if scanner.Err() == bufio.ErrTooLong {
			continue
		} else {
			break
		}

	}
	return
}

func (this *DocumentParser) StartTomita() (err error) {
	//this.locker.Lock()
	//defer this.locker.Unlock()

	if this.Cmd != nil && this.Cmd.Process != nil {
		this.Cmd.Process.Kill()
	}

	this.lastError = ""
	this.documentOffset = 0
	this.Cmd = nil
	this.bodyReader = NewReadersQueue()
	this.Stderr = nil
	this.Stdout = nil

	os.Chdir("parser")
	defer os.Chdir("..")
	//this.Cmd = exec.Command("tomitaparser.exe", "config.proto")
	this.Cmd = exec.Command("parser.bat", "config.proto")
	this.Cmd.Stdin = this.bodyReader
	this.Stdout, err = this.Cmd.StdoutPipe()
	if err != nil {
		return err
	}
	//this.Cmd.Stderr = os.Stderr
	this.Stderr, err = this.Cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = this.Cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	go this.ParseStderr()
	return err
}

func CreateDocumentParser() (this *DocumentParser, err error) {
	this = &DocumentParser{}

	this.Lock()
	defer this.Unlock()

	err = this.StartTomita()

	// this.Stdin.Write([]byte("АСЕПТОЛИН 90% Р-Р Д/НАРУЖ ПРИМЕНЕНИЯ 100МЛ ФЛАК - ФАРМАЦЕВТИЧЕСКИЙ КОМБИНАТ\n"))
	// scanner := bufio.NewScanner(this.Stdout)
	// scanner.Scan()

	// for scanner.Scan() {
	// 	log.Println(scanner.Text())
	// 	if err = scanner.Err(); err != nil {
	// 		log.Printf("reading standard input:", err)
	// 	}
	// }
	return this, err
}

func (doc *Document) PrintDocument() []byte {

	type Drug struct {
		Line          int64  `xml:"Line,attr"`
		Name          *Val   `xml:"Name"`
		Form          *[]Val `xml:"Form"`
		Dosage        *[]Val `xml:"Dosage"`
		TotalQuantity *[]Val `xml:"TotalQuantity"`
	}

	var drug Drug
	// line, err := inputReader.GetLine(doc.DocumentIndex)
	// if err != nil {
	// 	log.Panicf("Error while reading lead for drug, %v\n", err)
	// }
	drug.Line = doc.DocumentIndex
	drug.Name = &doc.Name
	drug.Dosage = &doc.Dosage
	drug.Form = &doc.Form
	drug.TotalQuantity = &doc.TotalQuantity

	b, err := xml.MarshalIndent(drug, "", "  ")
	if err != nil {
		log.Panicln(err)
	}
	return b
}

func (this *DocumentParser) ParseFacts(input io.Reader,
	writer io.Writer,
	maxFileSize int64) (err error) {
	this.Lock()
	defer this.Unlock()

	log.Printf("Enter into ParseFacts")

	//must restart Tomita after at the end
	defer func() {
		func() {
			this.StartTomita()
		}()
	}()

	this.bodyReader.AppendReader(
		io.LimitReader(input, maxFileSize))

	//scanner2 := bufio.NewScanner(this.Stdout)
	//for scanner2.Scan() {
	//	totalFacts += 1
	//	log.Printf("got tomita out: %v\n", totalFacts) //scanner2.Text())
	//	if err = scanner2.Err(); err != nil {
	//		log.Printf("reading standard input:", err)
	//	}
	//}
	//log.Println("finished tomita read")
	//return nil

	var docsBuffer bytes.Buffer

	docsBuffer.Write([]byte("<?xml version='1.0' encoding='utf-8'?>\n<Drugs>\n"))

	decoder := xml.NewDecoder(this.Stdout)
	for {
		t, err := decoder.Token()
		if t == nil {
			if err == nil {
				log.Println("token nil")
				continue
			}
			if err == io.EOF {
				//log.Printf("Tomita stopped ex :%v\n", this.Cmd.Wait())
				//log.Printf("Tomita stopped :%v\n", this.lastError)
				if n, _ := input.Read([]byte{0}); n == 1 {
					docsBuffer.Write([]byte("<Error>Parsing is interrupted</Error>"))
				}
				break
			} else {
				//That is not good
				log.Printf("Tomita stopped :%v\n", this.lastError)
				docsBuffer.Write([]byte("<Error>"))
				xml.EscapeText(
					&docsBuffer,
					[]byte(this.lastError))
				docsBuffer.Write([]byte("</Error>"))
			}
			//log.Fatal(err)
		}

		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "document" {
				var d Document
				// decode a whole chunk of following XML into the
				decoder.DecodeElement(&d, &se)

				docsBuffer.Write(d.PrintDocument())
				docsBuffer.Write([]byte("\n"))
				//log.Printf("print doc: %v\n", d.Name.Value)
			}
		}
	}

	docsBuffer.Write([]byte("</Drugs>"))

	//finally write to output
	docsBuffer.WriteTo(writer)

	log.Println("stop2!!!")
	return nil
}
