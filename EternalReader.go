package main

import (
	"io"
	//	"log"
)

type ReadersQueue struct {
	reader io.Reader
	rChan  chan io.Reader
}

func (qr *ReadersQueue) Read(p []byte) (n int, err error) {
	if qr.reader == nil {
		qr.reader = <-qr.rChan
	}
	return qr.reader.Read(p)
}

func NewReadersQueue() (qr *ReadersQueue) {
	qr = &ReadersQueue{
		rChan: make(chan io.Reader),
	}
	return qr
}

func (qr *ReadersQueue) AppendReader(r io.Reader) {
	qr.rChan <- r
}
