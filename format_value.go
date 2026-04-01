package main

// FormatValue and related helpers are adapted from the tfk8s project
// (https://github.com/jrhouston/tfk8s), which in turn was derived from
// Terraform core's internal/repl FormatValue function.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// defaultDelimiter is "End Of Text" by convention
const defaultDelimiter = "EOT"

// formatValue formats a cty.Value in a way that resembles Terraform HCL syntax.
func formatValue(v cty.Value, indent int) string {
	if !v.IsKnown() {
		return "(known after apply)"
	}
	if v.IsMarked() {
		return "(sensitive)"
	}
	if v.IsNull() {
		return formatNullValue(v)
	}

	ty := v.Type()
	switch {
	case ty.IsPrimitiveType():
		return formatPrimitiveValue(v, indent)
	case ty.IsObjectType():
		return formatMappingValue(v, indent)
	case ty.IsTupleType():
		return formatSequenceValue(v, indent)
	case ty.IsListType():
		return fmt.Sprintf("tolist(%s)", formatSequenceValue(v, indent))
	case ty.IsSetType():
		return fmt.Sprintf("toset(%s)", formatSequenceValue(v, indent))
	case ty.IsMapType():
		return fmt.Sprintf("tomap(%s)", formatMappingValue(v, indent))
	}

	return fmt.Sprintf("%#v", v)
}

func formatNullValue(v cty.Value) string {
	ty := v.Type()
	switch {
	case ty == cty.DynamicPseudoType:
		return "null"
	case ty == cty.String:
		return "tostring(null)"
	case ty == cty.Number:
		return "tonumber(null)"
	case ty == cty.Bool:
		return "tobool(null)"
	case ty.IsListType():
		return fmt.Sprintf("tolist(null) /* of %s */", ty.ElementType().FriendlyName())
	case ty.IsSetType():
		return fmt.Sprintf("toset(null) /* of %s */", ty.ElementType().FriendlyName())
	case ty.IsMapType():
		return fmt.Sprintf("tomap(null) /* of %s */", ty.ElementType().FriendlyName())
	default:
		return fmt.Sprintf("null /* %s */", ty.FriendlyName())
	}
}

func formatPrimitiveValue(v cty.Value, indent int) string {
	switch v.Type() {
	case cty.String:
		if formatted, isMultiline := formatMultilineString(v, indent); isMultiline {
			return formatted
		}
		return strconv.Quote(v.AsString())
	case cty.Number:
		bf := v.AsBigFloat()
		return bf.Text('f', -1)
	case cty.Bool:
		if v.True() {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%#v", v)
}

func formatMultilineString(v cty.Value, indent int) (string, bool) {
	str := v.AsString()
	lines := strings.Split(str, "\n")
	if len(lines) < 2 {
		return "", false
	}

	operator := "<<"
	if indent > 0 {
		operator = "<<-"
	}

	delimiter := defaultDelimiter

OUTER:
	for {
		for _, line := range lines {
			if strings.TrimSpace(line) == delimiter {
				delimiter = delimiter + "_"
				continue OUTER
			}
		}
		break
	}

	var buf strings.Builder
	buf.WriteString(operator)
	buf.WriteString(delimiter)
	for _, line := range lines {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(line)
	}
	buf.WriteByte('\n')
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteString(delimiter)

	return buf.String(), true
}

func formatMappingValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	prevComplex := false
	buf.WriteByte('{')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		k, v := it.Element()
		complex := isComplexType(v)

		// Add a blank line to visually separate groups: before complex values,
		// and before scalars that follow a complex value.
		if count > 1 && (complex || prevComplex) {
			buf.WriteByte('\n')
		}

		prevComplex = complex

		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(formatValue(k, indent))
		buf.WriteString(" = ")
		buf.WriteString(formatValue(v, indent))
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte('}')
	return buf.String()
}

func formatSequenceValue(v cty.Value, indent int) string {
	var buf strings.Builder
	count := 0
	buf.WriteByte('[')
	indent += 2
	for it := v.ElementIterator(); it.Next(); {
		count++
		_, v := it.Element()
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
		formattedValue := formatValue(v, indent)
		buf.WriteString(formattedValue)
		if strings.HasSuffix(formattedValue, defaultDelimiter) {
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat(" ", indent))
		}
		buf.WriteByte(',')
	}
	indent -= 2
	if count > 0 {
		buf.WriteByte('\n')
		buf.WriteString(strings.Repeat(" ", indent))
	}
	buf.WriteByte(']')
	return buf.String()
}

// escapeShellVars escapes ${} sequences to prevent Terraform interpolation.
// Adapted from tfk8s (https://github.com/jrhouston/tfk8s).
func escapeShellVars(s string) string {
	r := regexp.MustCompile(`(\${.*?)`)
	return r.ReplaceAllString(s, `$$$1`)
}

// snakify converts "a-String LIKE this" to "a_string_like_this".
// Adapted from tfk8s (https://github.com/jrhouston/tfk8s).
func snakify(s string) string {
	re := regexp.MustCompile(`\W`)
	return strings.ToLower(re.ReplaceAllString(s, "_"))
}

// camelToSnake converts a CamelCase string to snake_case.
// e.g. "CustomResourceDefinition" → "custom_resource_definition"
func camelToSnake(s string) string {
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

// isComplexType returns true if the cty.Value is a non-primitive type
// (object, tuple, list, set, map).
func isComplexType(v cty.Value) bool {
	if v.IsNull() || !v.IsKnown() {
		return false
	}
	ty := v.Type()
	return ty.IsObjectType() || ty.IsTupleType() || ty.IsListType() || ty.IsSetType() || ty.IsMapType()
}
