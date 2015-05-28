package toner

type content struct {
	f          file
	err        error
	rawContent []byte
	ctype      *ctype
	tags       []*tag
}

type ctype struct {
}

type tag struct {
}
