package text

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

const (
	MinNonSpecialAscii = 32
	MaxNonSpecialAscii = 126
	LetterPageHeight   = 841.89
	LetterPageWidth    = 595.28
)

func ParseXYValues(input string) (x float64, y float64, parseErr error) {
	XYParser := regexp.MustCompile(`^\s*(?P<x>-?\d+(\.\d+)?)\s+(?P<y>-?\d+(\.\d+)?)\s*$`)
	XIndex := XYParser.SubexpIndex("x")
	YIndex := XYParser.SubexpIndex("y")
	match := XYParser.FindStringSubmatch(input)
	if match == nil {
		parseErr = fmt.Errorf("could not parse '%s' into X Y values", input)
		return
	}
	x, parseErr = strconv.ParseFloat(match[XIndex], 64)
	y, parseErr = strconv.ParseFloat(match[YIndex], 64)
	return
}

func ParseSingleValue(input string) (value float64, parseErr error) {
	valueParser := regexp.MustCompile(`^\s*(?P<value>-?\d+(\.\d+)?)\s*$`)
	valueIndex := valueParser.SubexpIndex("value")
	match := valueParser.FindStringSubmatch(input)
	if match == nil {
		parseErr = fmt.Errorf("could not parse '%s' into float value", input)
		return
	}
	value, parseErr = strconv.ParseFloat(match[valueIndex], 64)
	return
}

func GetFloatParams(paramString string) (params []float64, parseErr error) {
	var num float64
	numberRegex := regexp.MustCompile(`(?P<param>-?[\d.]+)`)
	paramIndex := numberRegex.SubexpIndex("param")
	matches := numberRegex.FindAllStringSubmatch(paramString, -1)
	if matches == nil {
		parseErr = fmt.Errorf("failed to parse float params from '%s'", paramString)
		return
	} else {
		for _, match := range matches {
			if num, parseErr = strconv.ParseFloat(match[paramIndex], 64); parseErr != nil {
				return
			}
			params = append(params, num)
		}
	}
	return
}

func ParseTextFields(opString string) (textCharacters []*ShowChars) {
	outBuff := strings.Builder{}
	var err error
	adjustRegex := regexp.MustCompile(`[\d.\-]+$`)
	openParen := "("[0]
	closeParen := ")"[0]
	backslash := "\\"[0]
	lastByte := uint8(0)
	b := uint8(0)
	adjust := float64(0)
	insideParen := false
	for i := range opString {
		lastByte = b
		b = opString[i]

		// Open parenthesis is like an open quote and signifies the beginning of some text
		if b == openParen && lastByte != backslash {
			// If there as a float prior to the open paren, it's a horizontal adjustment--save it to adjust
			match := adjustRegex.FindString(opString[:i])
			if match == "" {
				adjust = 1
			} else {
				adjust, err = strconv.ParseFloat(match, 64)
				if err != nil {
					log.Warnf("unable to parse TJ adjustment float from '%s'", opString[:i])
					adjust = 1
				}
			}
			insideParen = true
			outBuff.Reset()
			continue
		}
		// A close paren is ending some text, create new line chars with the text we captured since the open
		// paren along with the adjustment (or use 1 as default)
		if b == closeParen && lastByte != backslash {
			insideParen = false
			if outBuff.Len() > 0 {
				charSet := ShowChars{
					Text:        outBuff.String(),
					HorizAdjust: adjust,
				}
				textCharacters = append(textCharacters, &charSet)
				adjust = 0
			}
			continue
		}
		// If we have a backslash w/o a preceding one, it's an escape so continue w/o writing it to buffer
		if b == backslash && lastByte != backslash {
			continue
		}
		if insideParen {
			outBuff.WriteByte(b)
		}
	}
	return
}

func StringToHexDump(input string) string {
	var hexChars []string
	for _, char := range input {
		hexChars = append(hexChars, fmt.Sprintf("%02X", char))
	}
	return strings.Join(hexChars, " ")
}
