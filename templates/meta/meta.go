// Package meta defines shared template metadata.
package meta

import "time"

const (
	AppTitle = "App"
)

var (
	Year int
)

func init() {
	Year = time.Now().Year()
}
