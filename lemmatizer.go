package main

import (
	"bufio"
	"io"
	"log"
	"regexp"
)

type Lemmatizer struct {
	re0     *regexp.Regexp
	re1     *regexp.Regexp
	re2     *regexp.Regexp
	re3     *regexp.Regexp
	re4     *regexp.Regexp
	re5     *regexp.Regexp
	re6     *regexp.Regexp
	r       *bufio.Reader
	rChan   chan *bufio.Reader
	readErr error
	out     []byte
}

func (l *Lemmatizer) Read(p []byte) (copied int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	copied = 0
	for {
		// Copy leftover output from last decode.
		if len(l.out) > 0 {
			n := copy(p[copied:], l.out)
			copied = copied + n
			l.out = l.out[n:]
			if len(l.out) > 0 { //input buffer is not enough
				return copied, nil
			}
		}

		// Out of input, out of decoded output.  Check errors.
		if l.readErr != nil {
			return copied, nil //never return eof
		}

		if l.r == nil {
			l.r = <-l.rChan
			if l.r == nil {
				return 0, io.EOF
			}
			l.readErr = nil
		}

		line, err := l.r.ReadString(byte('\n'))
		l.readErr = err
		if len(line) > 0 {
			l.out = []byte(l.Lemmatize(line)) // + "\n"
		}
		if l.readErr != nil {
			log.Printf("lemma inp err: %v\n", l.readErr)
			l.out = append(l.out, []byte("\ntomita_eof tomita_eof\n")...)
			l.r = nil
		}
	}

}

// s/\. /_ /gi;
// s/([а-яА-Я])\.([^\.])/$1_ $2/gi;
// s/(\d)([а-яА-Я%]{1,3})/$1 $2/gi; # 150мкг -> 150 мкг
// s/(\d),(\d)/$1.$2/gi; # number
// s/\b([NXХ])(\d)/$1 $2/g; # N30 -> N 30
// s/([а-яА-Я])\.(\d)/$1_ $2/gi;

func CreateLemmatizer() (this *Lemmatizer) {
	this = &Lemmatizer{
		re0:   regexp.MustCompile(`(?i)\. `),
		re1:   regexp.MustCompile(`(?i)([а-яА-Я])\.([^\.])`),
		re2:   regexp.MustCompile(`(?i)(\d)([а-яА-Я%]{1,3})`),
		re3:   regexp.MustCompile(`(?i)(\d),(\d)`),
		re4:   regexp.MustCompile(`(?i)\b([NXХ])(\d)`),
		re5:   regexp.MustCompile(`(?i)([а-яА-Я])\.(\d)`),
		re6:   regexp.MustCompile(`(?i)tomita_eof`),
		rChan: make(chan *bufio.Reader),
	}
	return this
}

func (l *Lemmatizer) SetInput(r io.Reader) {
	l.rChan <- bufio.NewReader(r)
}

func (this *Lemmatizer) Lemmatize(inp string) string {
	inp = this.re0.ReplaceAllString(inp, "_ ")
	inp = this.re1.ReplaceAllString(inp, "$1_ $2")
	inp = this.re2.ReplaceAllString(inp, "$1 $2")
	inp = this.re3.ReplaceAllString(inp, "$1.$2")
	inp = this.re4.ReplaceAllString(inp, "$1 $2")
	inp = this.re5.ReplaceAllString(inp, "$1_ $2")
	inp = this.re6.ReplaceAllString(inp, "tomita eof")
	return inp
}
