package toner

type contentGenImg struct {
	c *content
}

type img struct {
	src  string
	ext  string
	w    uint
	h    uint
	crop imgCrop
}

type imgCrop int

const (
	cropLeft imgCrop = iota
	cropCentered
	cropLen
)

var imgExts = []string{
	".jpg",
	".jpeg",
	".png",
	".gif",
}

func (gi contentGenImg) getGenerator(c *content, ext string) interface{} {
	for _, e := range imgExts {
		if ext == e {
			return contentGenImg{
				c: c,
			}
		}
	}

	return nil
}

func (gi contentGenImg) generatePage() (string, error) {
	// c := gi.c
	// s := c.cs.s
	return "", nil
}

func (contentGenImg) humanName() string {
	return "image"
}

func (gi contentGenImg) scale(img img) (dstPath string, err error) {
	dstPath, alreadyClaimed, err := gi.c.claimStaticDest("img", img.ext)
	if alreadyClaimed || err != nil {
		return
	}

	return
}
