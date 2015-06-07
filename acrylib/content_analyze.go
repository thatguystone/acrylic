package acrylib

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type contentAnalyze struct {
	cfg   *Config
	gen   contentGenWrapper
	deets *contentDetails

	content string
}

const summarySplit = "<!--more-->"

var summarySentenceRe = regexp.MustCompile(`[^\.!\?]+[\.!\?]+`)

func (ca *contentAnalyze) analyze() {
	if ca.deets.analyzed {
		return
	}

	// This is a heavy operation, block any others
	ca.deets.mtx.Lock()

	if !ca.deets.analyzed {
		ca.extractWordCountAndSummary()
		ca.doStats()
		ca.deets.analyzed = true
	}

	ca.deets.mtx.Unlock()
}

func (ca *contentAnalyze) extractWordCountAndSummary() {
	ca.content = ca.gen.getContent()

	if len(ca.deets.summary) == 0 {
		splitAt := strings.Index(ca.content, summarySplit)
		if splitAt > -1 {
			ca.deets.summary = ca.dropTags(ca.content[:splitAt], nil)
		}
	}

	cleaned := ca.dropTags(ca.content, func(wordCount int, buff []byte) {
		ca.deets.wordCount += wordCount

		if len(ca.deets.summary) == 0 && ca.deets.wordCount >= ca.cfg.SummaryWords {
			matches := summarySentenceRe.FindAll(buff, -1)

			summaryLen := 0
			summary := bytes.Buffer{}
			for i, m := range matches {
				if i > 0 {
					summary.Write([]byte(" "))
				}

				m = bytes.TrimSpace(m)
				summary.Write(m)
				summaryLen += bytes.Count(m, []byte(" ")) + 1

				if summaryLen >= ca.cfg.SummaryWords {
					break
				}
			}

			ca.deets.summary = summary.String()
		}
	})

	if len(ca.deets.summary) == 0 {
		ca.deets.summary = cleaned
	}
}

func (ca *contentAnalyze) dropTags(in string, textCb func(int, []byte)) string {
	b := bytes.Buffer{}
	z := html.NewTokenizer(strings.NewReader(in))

	errd := false
	for !errd {
		switch z.Next() {
		case html.ErrorToken:
			errd = true

		case html.TextToken:
			words := bytes.Fields(z.Text())
			b.Write(bytes.Join(words, []byte(" ")))

			if textCb != nil {
				textCb(len(words), b.Bytes())
			}
		}
	}

	return b.String()
}

func (ca *contentAnalyze) doStats() {
	ca.deets.fuzzyWordCount = ((ca.deets.wordCount + 100) / 100) * 100
}
