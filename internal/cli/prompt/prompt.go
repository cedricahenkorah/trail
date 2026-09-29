package prompt

import "github.com/charmbracelet/huh"

var draculaTheme = huh.ThemeDracula()

func Run(field huh.Field) error {
	return field.WithTheme(draculaTheme).Run()
}
