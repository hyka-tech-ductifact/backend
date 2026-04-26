package valueobjects

import (
	"fmt"
	"strings"
)

// Locale represents a validated BCP 47 language tag (e.g. "en", "es").
// It is the single source of truth for supported languages.
type Locale struct {
	value string
}

// Supported locale constants.
var (
	LocaleEN = Locale{value: "en"}
	LocaleES = Locale{value: "es"}
)

// DefaultLocale is the fallback when none is specified.
var DefaultLocale = LocaleEN

// supported lists every locale the application can handle.
// Add new locales here — everything else derives from this slice.
var supported = []Locale{LocaleEN, LocaleES}

// NewLocale validates a raw string and returns a Locale.
// Returns ErrInvalidLocale if the value is not in the supported list.
func NewLocale(raw string) (Locale, error) {
	for _, l := range supported {
		if raw == l.value {
			return l, nil
		}
	}
	return Locale{}, ErrInvalidLocale()
}

// String returns the BCP 47 tag (e.g. "en").
func (l Locale) String() string {
	return l.value
}

// SupportedLocales returns a copy of all supported locale strings.
func SupportedLocales() []string {
	out := make([]string, len(supported))
	for i, l := range supported {
		out[i] = l.value
	}
	return out
}

// ErrInvalidLocale builds a descriptive error from the supported list.
func ErrInvalidLocale() error {
	return fmt.Errorf("invalid locale: must be one of: %s", strings.Join(SupportedLocales(), ", "))
}
