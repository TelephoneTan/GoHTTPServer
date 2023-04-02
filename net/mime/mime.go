package mime

type Type = string

const (
	TextPlainUTF8 Type = "text/plain; charset=UTF-8"
	JSON          Type = "application/json"
	HTML          Type = "text/html"
)
