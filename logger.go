package acrylic

// A Logger is used for all acrylic logging
type Logger interface {
	Log(msg string)
	Error(err error, msg string)
}
