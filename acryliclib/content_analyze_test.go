package acryliclib

import (
	"path/filepath"
	"testing"

	"github.com/thatguystone/assert"
)

func testContentAnalyze(t *testing.T, cfg *Config, content string) *contentDetails {
	tt := testNew(t, true, cfg, testFile{
		p:  "content/render.html",
		sc: content,
	})
	defer tt.cleanup()

	c := tt.lastSite.cs.srcs[filepath.Join(tt.cfg.Root, "content/render.html")]
	c.analyze()

	return &c.deets
}

func TestContentAnalyzeShortSummary(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	loremIpsum := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque metus enim, efficitur ut convallis id, venenatis feugiat urna. Pellentesque pulvinar iaculis ipsum ac bibendum. Suspendisse sit amet ipsum eget arcu dapibus pulvinar. Praesent at tortor erat. Curabitur feugiat viverra orci non commodo. Nunc dictum mauris ut ultrices tincidunt. Donec sodales leo nunc, et faucibus nisi vehicula et. Nunc posuere orci quam, nec molestie nulla consectetur id."

	cfg := testConfig()
	cfg.SummaryWords = 10
	deets := testContentAnalyze(t, cfg, loremIpsum)
	a.Equal(
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque metus enim, efficitur ut convallis id, venenatis feugiat urna.`,
		deets.summary)
	a.Equal(66, deets.wordCount)
	a.Equal(100, deets.fuzzyWordCount)
}

func TestContentAnalyzeLongSummary(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	deets := testContentAnalyze(t, nil, longLoremIpsum)
	a.Equal(
		`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque metus enim, efficitur ut convallis id, venenatis feugiat urna. Pellentesque pulvinar iaculis ipsum ac bibendum. Suspendisse sit amet ipsum eget arcu dapibus pulvinar. Praesent at tortor erat. Curabitur feugiat viverra orci non commodo. Nunc dictum mauris ut ultrices tincidunt. Donec sodales leo nunc, et faucibus nisi vehicula et. Nunc posuere orci quam, nec molestie nulla consectetur id. Aliquam erat volutpat. Suspendisse commodo metus sit amet dolor vehicula, nec faucibus orci egestas.`,
		deets.summary)
	a.Equal(393, deets.wordCount)
	a.Equal(400, deets.fuzzyWordCount)
}

func TestContentAnalyzeSummarySplit(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	loremIpsum := "Lorem ipsum dolor sit amet<!--more-->, consectetur adipiscing elit. Pellentesque metus enim, efficitur ut convallis id, venenatis feugiat urna. Pellentesque pulvinar iaculis ipsum ac bibendum."

	deets := testContentAnalyze(t, nil, loremIpsum)
	a.Equal(
		`Lorem ipsum dolor sit amet`,
		deets.summary)
}

func BenchmarkContentAnalyzeLongSummary(b *testing.B) {
	tt := testNew(b, true, nil, testFile{
		p:  "content/render.html",
		sc: longLoremIpsum,
	})
	defer tt.cleanup()

	c := tt.lastSite.cs.srcs[filepath.Join(tt.cfg.Root, "content/render.html")]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.deets = contentDetails{}
		c.analyze()
	}
}

const longLoremIpsum = `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque metus enim, efficitur ut convallis id, venenatis feugiat urna. Pellentesque pulvinar iaculis ipsum ac bibendum. Suspendisse sit amet ipsum eget arcu dapibus pulvinar. Praesent at tortor erat. Curabitur feugiat viverra orci non commodo. Nunc dictum mauris ut ultrices tincidunt. Donec sodales leo nunc, et faucibus nisi vehicula et. Nunc posuere orci quam, nec molestie nulla consectetur id.</p>

<p>Aliquam erat volutpat. Suspendisse commodo metus sit amet dolor vehicula, nec faucibus orci egestas. Praesent commodo fermentum elementum. Cras in elit bibendum, ultricies eros ut, scelerisque augue. Phasellus at enim tortor. Mauris auctor lorem dolor, vel ultricies velit euismod at. Donec eu finibus mauris. Duis lectus lectus, pharetra nec mi eget, pellentesque consequat arcu. Sed nibh nulla, congue vestibulum ullamcorper ut, bibendum nec nulla. Nullam et mattis quam. Sed sit amet nisl nec eros commodo fringilla. Integer nec lacus in magna scelerisque cursus. Etiam a dolor consectetur, vehicula purus id, convallis purus. Phasellus feugiat augue ac nunc dictum, sit amet auctor turpis convallis. Nullam ultrices sapien nec sagittis lobortis. Fusce lorem turpis, aliquet eu lobortis ac, volutpat vel velit.</p>

<p>Morbi sit amet turpis ac neque fermentum scelerisque. Vestibulum metus diam, laoreet sed viverra a, venenatis sit amet diam. Mauris aliquam consectetur metus sit amet facilisis. Praesent rhoncus lectus eu orci molestie mollis. Aliquam justo augue, tincidunt malesuada ligula non, pellentesque tempor arcu. Phasellus vestibulum arcu nec dapibus porta. Nam ultricies convallis nibh at dictum. Aenean congue pharetra auctor. Nam porttitor dui vel nisl hendrerit, ac venenatis mi auctor. Fusce eget massa sit amet odio vestibulum egestas eu et lectus. In et leo nisl.</p>

<p>Ut sit amet malesuada dui, id placerat orci. Integer eleifend eros metus, sed fringilla leo suscipit at. Nunc ac tortor auctor, tincidunt augue vel, hendrerit nunc. Donec vel facilisis massa, ut molestie erat. Maecenas id viverra velit, quis bibendum diam. Nulla efficitur dui felis, vitae placerat nulla semper ut. Aliquam posuere nisl id posuere sagittis. Integer convallis libero ut erat ultricies, vel molestie lectus tincidunt.</p>

<p>Nam in nulla est. Nam eget quam tempus, facilisis orci eget, porttitor nulla. Maecenas sit amet erat diam. In sit amet ullamcorper libero. Aenean congue sem massa, at ullamcorper dui accumsan ut. Proin tempus, magna molestie bibendum malesuada, mauris libero vulputate odio, non aliquet ante nulla vel enim. Phasellus dui felis, suscipit ac rhoncus sit amet, tempor eget dolor.</p>`
