// Package i18n registers the application's locales.
package i18n

import "github.com/tavocg/go-i18n"

var locales = map[string]i18n.Locale{}

func Locales() map[string]i18n.Locale {
	return locales
}
