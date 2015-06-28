package acrylib

import (
	"bytes"
	"io"
	"os/exec"
)

const pygmentsBin = "pygmentize"

func canHighlight() bool {
	_, err := exec.LookPath(pygmentsBin)
	return err == nil
}

func highlight(lang string, out io.Writer, code []byte) error {
	cmd := exec.Command(pygmentsBin, "-f", "html", "-l", lang)
	cmd.Stdin = bytes.NewReader(code)
	cmd.Stdout = out

	return cmd.Run()
}
