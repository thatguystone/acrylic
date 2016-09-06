package imgs

import (
	"sort"
	"sync"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/state"
)

type Imgs struct {
	st *state.S

	mtx   sync.Mutex
	all   imgSlice
	imgs  map[string]*Img
	byCat map[string]imgSlice
}

type imgSlice []*Img

func New(st *state.S) *Imgs {
	return &Imgs{
		st:    st,
		imgs:  map[string]*Img{},
		byCat: map[string]imgSlice{},
	}
}

func (is *Imgs) Load(f file.F, isContent bool) error {
	img, err := newImg(is.st, f, isContent)
	if err == nil {
		is.mtx.Lock()

		is.all = append(is.all, img)
		is.imgs[f.Src] = img

		cat := is.byCat[f.Cat]
		cat = append(cat, img)
		is.byCat[f.Cat] = cat

		is.mtx.Unlock()
	}

	return err
}

func (is *Imgs) AllLoaded() {
	is.st.Run.Do(func() {
		sort.Sort(is.all)
	})

	for _, ims := range is.byCat {
		func(ims imgSlice) {
			is.st.Run.Do(func() {
				sort.Sort(ims)
			})
		}(ims)
	}
}

func (is *Imgs) All(inGallery bool) (imgs []string) {
	for _, img := range is.all {
		if inGallery && !img.inGallery {
			continue
		}

		abs := "/" + img.URL
		imgs = append(imgs, abs)
	}

	return
}

func (is *Imgs) Get(path string) *Img {
	return is.imgs[path]
}

func (is imgSlice) Len() int           { return len(is) }
func (is imgSlice) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is imgSlice) Less(i, j int) bool { return is[i].SortName > is[j].SortName }
