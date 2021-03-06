//originally by Andrew Gerrand
package main

import (
	//	"bufio"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type gzipRequestReader struct {
	io.Reader
	io.ReadCloser
}

func (r gzipRequestReader) Read(b []byte) (int, error) {
	return r.Reader.Read(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Println(err)
			} else {
				defer r.Body.Close()
				r.Body = gzipRequestReader{Reader: gz, ReadCloser: r.Body}
			}
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func makeGzipWriterHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			rPipe, wPipe := io.Pipe()
			newWriter := gzip.NewWriter(wPipe)
			oldBody := r.Body
			go func() {
				_, err := io.Copy(newWriter, oldBody)
				oldBody.Close()
				newWriter.Close()
				wPipe.Close()

				if err != nil {
					log.Fatalf("makeGzipWriterHandler error: %v", err)
				}
			}()

			r.Body = rPipe
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}
