package crawl

import (
	"strings"
	"unicode"
)

type srcSetTransform struct {
	c      *Content
	srcSet srcSet
	cs     []*Content
}

func newSrcSetTransform(cr *Crawler, c *Content, val string) srcSetTransform {
	srcSet := parseSrcSet(val)

	tf := srcSetTransform{
		c:      c,
		srcSet: srcSet,
		cs:     make([]*Content, len(srcSet)),
	}

	for i, src := range srcSet {
		tf.cs[i] = cr.GetRel(c, src.url)
	}

	return tf
}

func (tf srcSetTransform) get() string {
	for i := range tf.srcSet {
		tf.srcSet[i].url = tf.c.GetLinkTo(tf.cs[i], tf.srcSet[i].url)
	}

	return tf.srcSet.String()
}

type srcSet []imgSrc

type imgSrc struct {
	url         string
	descriptors []string
}

func parseSrcSet(s string) (ss srcSet) {
	type parseState int
	const (
		stateStart parseState = iota
		stateInURL
		stateInDescriptor
		stateInParens
	)

	n := strings.Count(s, ",")
	if n > 0 {
		ss = make(srcSet, 0, n)
	}

	var (
		start = 0
		state = stateStart
		src   = imgSrc{}
		add   = func() {
			ss = append(ss, src)
			src = imgSrc{}
		}
		setURL = func(end int) {
			if start < end {
				src.url = strings.TrimRight(s[start:end], ",")
			}
		}
		addDescriptor = func(end int) {
			if start < end {
				src.descriptors = append(src.descriptors, s[start:end])
			}
		}
	)

	// Based on:
	//  * https://html.spec.whatwg.org/multipage/images.html#parsing-a-srcset-attribute
	//  * https://hg.mozilla.org/mozilla-central/file/718c237332ba/servo/components/script/dom/htmlimageelement.rs#l1081
	for i, r := range s {
		switch state {
		case stateStart:
			if !unicode.IsSpace(r) && r != ',' {
				start = i
				state = stateInURL
			}

		case stateInURL:
			if !unicode.IsSpace(r) {
				continue
			}

			setURL(i)
			if s[i-1] == ',' {
				add()
				state = stateStart
			} else {
				start = i + 1
				state = stateInDescriptor
			}

		case stateInDescriptor:
			switch {
			case unicode.IsSpace(r):
				addDescriptor(i)
				start = i + 1

			case r == ',':
				addDescriptor(i)
				add()
				state = stateStart

			case r == '(':
				state = stateInParens

			default:
				// No changes, just keep moving
			}

		case stateInParens:
			if r == ')' {
				state = stateInDescriptor
			}
		}
	}

	// EOF cases
	switch state {
	case stateInURL:
		setURL(len(s))

	case stateInDescriptor, stateInParens:
		addDescriptor(len(s))
	}

	if src.url != "" {
		add()
	}

	return
}

func (ss srcSet) String() string {
	var b strings.Builder

	for i, src := range ss {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(src.url)

		for _, d := range src.descriptors {
			b.WriteString(" ")
			b.WriteString(d)
		}
	}

	return b.String()
}
