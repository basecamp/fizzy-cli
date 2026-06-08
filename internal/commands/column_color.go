package commands

import (
	"fmt"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

type columnColor struct {
	Name  string
	Value string
}

var columnColors = []columnColor{
	{Name: "blue", Value: "var(--color-card-default)"},
	{Name: "gray", Value: "var(--color-card-1)"},
	{Name: "tan", Value: "var(--color-card-2)"},
	{Name: "yellow", Value: "var(--color-card-3)"},
	{Name: "lime", Value: "var(--color-card-4)"},
	{Name: "aqua", Value: "var(--color-card-5)"},
	{Name: "violet", Value: "var(--color-card-6)"},
	{Name: "purple", Value: "var(--color-card-7)"},
	{Name: "pink", Value: "var(--color-card-8)"},
}

var columnColorNamesHelp = func() string {
	names := make([]string, len(columnColors))
	for i, color := range columnColors {
		names[i] = color.Name
	}
	return strings.Join(names, ", ")
}()

func normalizeColumnColor(input string) (string, error) {
	color := strings.TrimSpace(input)
	if color == "" {
		return "", nil
	}

	for _, valid := range columnColors {
		if strings.EqualFold(color, valid.Name) || color == valid.Value {
			return valid.Value, nil
		}
	}

	return "", errors.NewInvalidArgsError(fmt.Sprintf("--color must be one of: %s; or a supported API color value like %s (got %q)", columnColorNamesHelp, columnColors[0].Value, input))
}
