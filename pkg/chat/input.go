package chat

import "os"

type Input struct {
	Query string
	File  *os.File
}
