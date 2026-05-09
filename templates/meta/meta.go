// Package meta defines shared template metadata.
package meta

import "time"

const (
	AppTitle = "MyServer"
)

var (
	Year int
)

func init() {
	Year = time.Now().Year()
}
