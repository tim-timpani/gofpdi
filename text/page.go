package text

// render.go - the heart of text rendering for a given PDF page and a beast.  Seems like a lot of work playing
// with fonts, character widths, etc.  1) we need an accurate placement of the text for plain text files
// to avoid what so many other utilities tend to do.  In PDF files, parts of a sentence can be in different text
// blocks or the end punctuation can be in its own text block with its own x/y coordinates which makes placing
// it correctly even in plain text, a challenge. 2) by maintaining the font information, it leaves the door open
// for other file formats that have formatted text.

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	rowInsertionPrecision = 0.0000001
	maxRowInsertVariance  = 50.0
	minRowInsertBoundary  = 1
	maxRowInsertBoundary  = LetterPageWidth
)

type PageRender struct {
	PageNumber           int
	LineMatrix           *LinearMatrix
	TextMatrix           *LinearMatrix
	TransformationMatrix *LinearMatrix
	LineItems            []*ShowOperation
	Fonts                map[string]*FontDefinition
	Leading              float64
	CharSpacing          float64
	WordSpacing          float64
	Scale                float64
	FontSize             float64
	RenderMode           int
	Rise                 float64
	Knockout             float64
	FontName             string
}

func NewPageRender(pageNumber int, fonts map[string]*FontDefinition) *PageRender {
	return &PageRender{
		PageNumber:           pageNumber,
		LineMatrix:           NewDefaultMatrix(),
		TextMatrix:           NewDefaultMatrix(),
		TransformationMatrix: NewDefaultMatrix(),
		Fonts:                fonts,
	}
}

// AddTextLine creates a new line of text, copying the current values from the block (in that moment in time)
func (r *PageRender) addLineItem(chars []*ShowChars) error {

	font, found := r.Fonts[r.FontName]
	if !found {
		return fmt.Errorf("font '%s' not found in available page fonts", r.FontName)
	}

	line := ShowOperation{
		chars:       chars,
		PageNumber:  r.PageNumber,
		StartX:      r.TextMatrix.GetOffsetX(),
		StartY:      r.TextMatrix.GetOffsetY(),
		FontSize:    r.FontSize,
		Font:        font,
		Leading:     r.Leading,
		CharSpacing: r.CharSpacing,
		WordSpacing: r.WordSpacing,
		Scale:       r.Scale,
		RenderMode:  r.RenderMode,
		Rise:        r.Rise,
		Knockout:    r.Knockout,
	}
	lineWidth, err := line.GetWidth()
	if err != nil {
		return err
	}
	beforeX := r.TextMatrix.GetOffsetX()
	beforeY := r.TextMatrix.GetOffsetY()
	r.TextMatrix.Translate(lineWidth, 0)
	log.Debugf("Added text '%s' x/y before=%f/%f after=%f/%f", line.GetText(true),
		beforeX, beforeY, r.TextMatrix.GetOffsetX(), r.TextMatrix.GetOffsetY())
	r.LineItems = append(r.LineItems, &line)
	return nil
}

// moveToStartOfNextLine satisfies Td and TD operators
func (r *PageRender) moveToStartOfNextLine(opString string, setLeading bool) error {
	if opString == "" {
		r.LineMatrix.SetOffsetY(r.LineMatrix.GetOffsetY() - r.Leading)
	} else {
		x, y, err := ParseXYValues(opString)
		if err != nil {
			return err
		}
		if setLeading {
			r.Leading = y
		}
		r.LineMatrix.Move(x, y)
	}
	r.TextMatrix = r.LineMatrix.Copy()
	return nil
}

// setTextLeading satisfies TL operator
func (r *PageRender) setTextLeading(opString string) error {
	value, err := ParseSingleValue(opString)
	if err != nil {
		return err
	}
	r.Leading = value
	return nil
}

// setWordSpacing satisfies Tw operator
func (r *PageRender) setWordSpacing(opString string) error {
	value, err := ParseSingleValue(opString)
	if err != nil {
		return err
	}
	r.WordSpacing = value
	return nil
}

// setCharSpacing satisfies Tc operator
func (r *PageRender) setCharSpacing(opString string) error {
	value, err := ParseSingleValue(opString)
	if err != nil {
		return err
	}
	r.CharSpacing = value
	return nil
}

// setScale satisfies Tz operator
func (r *PageRender) setScale(opString string) error {
	value, err := ParseSingleValue(opString)
	if err != nil {
		return err
	}
	r.Scale = value / 100
	return nil
}

// moveToStartOfNextLineAndAddText satisfies " operator
func (r *PageRender) moveToStartOfNextLineAndAddText(opString string) error {
	quoteParser := regexp.MustCompile(`^\s*(?P<word>[\d.\-]+)\s+(?P<char>[\d.\-]+)\s*(?P<text>.+)$`)
	wordIndex := quoteParser.SubexpIndex("word")
	charIndex := quoteParser.SubexpIndex("char")
	textIndex := quoteParser.SubexpIndex("text")
	match := quoteParser.FindStringSubmatch(opString)
	if match == nil {
		return fmt.Errorf("failed to parse operand string '%s' for \" operator", opString)
	}
	r.LineMatrix.SetOffsetY(r.LineMatrix.GetOffsetY() - r.Leading)
	r.TextMatrix = r.LineMatrix.Copy()
	if err := r.setWordSpacing(match[wordIndex]); err != nil {
		return err
	}
	if err := r.setCharSpacing(match[charIndex]); err != nil {
		return err
	}
	if err := r.addText(match[textIndex]); err != nil {
		return err
	}
	return nil
}

// setMatrix satisfies Tm operation and MUST set both text and line matrices
func (r *PageRender) setMatrix(opString string) error {
	params, err := GetFloatParams(opString)
	if err != nil {
		return err
	}
	if len(params) != 6 {
		return fmt.Errorf("received %d params from string '%s' expecting 6", len(params), opString)
	}
	r.TextMatrix.Set(params[0], params[1], params[2], params[3], params[4], params[5])
	r.LineMatrix.Set(params[0], params[1], params[2], params[3], params[4], params[5])
	return nil
}

func (r *PageRender) setTextFont(opString string) (parseErr error) {
	fontRegex := regexp.MustCompile(`^\s*(?P<name>\S+)\s+(?P<size>[\d.]+)\s*$`)
	nameIndex := fontRegex.SubexpIndex("name")
	sizeIndex := fontRegex.SubexpIndex("size")
	match := fontRegex.FindStringSubmatch(opString)
	if match == nil {
		log.Errorf("font parse failure opString='%s' %s regex='%s'", StringToHexDump(opString), opString, fontRegex.String())
		parseErr = fmt.Errorf("failed to parse font name and size from '%s'", opString)
		return
	}
	r.FontName = match[nameIndex]
	r.FontSize, parseErr = strconv.ParseFloat(match[sizeIndex], 64)
	return
}

// addText satisfies Tj operator and used for text portion of " operator
func (r *PageRender) addText(opString string) error {
	return r.addLineItem(ParseTextFields(opString))
}

// AddTextBlock - add a block of text to the page
func (r *PageRender) AddTextBlock(textBlock string) (blockErr error) {

	log.Debug("* * * * * * * * new text block * * * * * * * *")

	validRegex := regexp.MustCompile(`^BT(?P<contents>(?s).*?)ET$`)
	match := validRegex.FindStringSubmatch(textBlock)
	if match == nil {
		blockErr = fmt.Errorf("'%s' is not a valid text block", textBlock)
		return
	}
	contents := match[validRegex.SubexpIndex("contents")]
	if strings.Contains(contents, "BT") || strings.Contains(contents, "ET") {
		blockErr = errors.New("text block cannot contain another start or end text block")
		return
	}

	capitalT := "T"[0]
	singleQuote := "'"[0]
	doubleQuote := "\""[0]
	openParen := "("[0]
	closeParen := ")"[0]
	backSlash := "\\"[0]
	var operator string
	var operandString string
	var eof error
	blockContent := strings.NewReader(contents)
	opBuilder := strings.Builder{}
	lastByte := uint8(0)
	b := uint8(0)
	insideParen := false
	for blockContent.Len() > 0 {
		lastByte = b
		b, eof = blockContent.ReadByte()
		if eof != nil {
			break
		}
		opBuilder.WriteByte(b)
		if insideParen {
			if b == closeParen && lastByte != backSlash {
				insideParen = false
			}
			continue
		}
		if b == openParen && lastByte != backSlash {
			insideParen = true
			continue
		}

		if lastByte == capitalT {
			splitPoint := len(opBuilder.String()) - 2
			operator = opBuilder.String()[splitPoint:]
			operandString = opBuilder.String()[:splitPoint]
		} else if b == singleQuote || b == doubleQuote {
			splitPoint := len(opBuilder.String()) - 1
			operator = opBuilder.String()[splitPoint:]
			operandString = opBuilder.String()[:splitPoint]
		} else {
			continue
		}
		log.Debugf("operator='%s' - %s", operator, operandString)
		switch operator {
		case "Td":
			blockErr = r.moveToStartOfNextLine(operandString, false)
		case "TD":
			blockErr = r.moveToStartOfNextLine(operandString, true)
		case "Tm":
			blockErr = r.setMatrix(operandString)
		case "Tj", "TJ":
			blockErr = r.addText(operandString)
		case "T*":
			blockErr = r.moveToStartOfNextLine("", false)
		case "TL":
			blockErr = r.setTextLeading(operandString)
		case "Tc":
			blockErr = r.setCharSpacing(operandString)
		case "Tw":
			blockErr = r.setWordSpacing(operandString)
		case "Tz":
			blockErr = r.setScale(operandString)
		case "Tf":
			blockErr = r.setTextFont(operandString)
		case "'":
			blockErr = r.moveToStartOfNextLine("", false)
			blockErr = r.addText(operandString)
		case "\"":
			blockErr = r.moveToStartOfNextLineAndAddText(operandString)
		case "Tr", "Ts":
			// ignore
		default:
			blockErr = fmt.Errorf("unrecognized text operator '%s' buff=%s", operator, opBuilder.String())
		}
		if blockErr != nil {
			return
		}
		lastByte = 0
		opBuilder.Reset()
		operator = ""
		log.Debugf("after operation tmx=%f tmy=%f lmx=%f lmy=%f", r.TextMatrix.GetOffsetX(),
			r.TextMatrix.GetOffsetY(), r.LineMatrix.GetOffsetX(), r.LineMatrix.GetOffsetY())
	}
	return
}

// GetIndexedShowOps - returns an indexed map row number x column index for the placement of each show operation
// since we are waling through it anyway, we also gather the plain text.  We return both so that we can support
// other file formats besides plain text.
func (r *PageRender) GetIndexedShowOps() (showOps map[int]map[float64]*ShowOperation, text string, pageErr error) {

	showOps = make(map[int]map[float64]*ShowOperation)

	// build a map indexed by column and row
	var row map[float64]*ShowOperation
	var rowFound bool

	// TODO: Re-work this section
	// Need to be more deterministic about finding the nearest line based on the font size and x/y values

	// loop through the show operations building a 2D map for row number (int) and column (float64)
	// for spacing between rows, it's better to align text intended for the same row, so we'll reduce them
	// to an int. For spacing across a line, it's more important to be more precise as a sentence can be split
	// across different showing operands and even different text blocks--although that would be rare.
	for _, showOp := range r.LineItems {

		rowNumber := int(showOp.StartY / 10)
		mapKeyX := showOp.StartX

		// y position exists, just get the row
		if row, rowFound = showOps[rowNumber]; rowFound {
			row = showOps[rowNumber]
			// Place the text at the nearest x position
			mapKeyX, pageErr = InsertShowOpIntoRow(showOp, mapKeyX, row)

			// y position does not exist, add an empty map for the row
		} else {
			showOps[rowNumber] = make(map[float64]*ShowOperation)
			showOps[rowNumber][mapKeyX] = showOp
		}
	}

	// build a sorted index for Y
	var indicesY []int
	for key := range showOps {
		indicesY = append(indicesY, key)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indicesY)))

	// Loop through Y index, sorting each row
	renderBuffer := strings.Builder{}
	var indicesX []float64
	for _, posY := range indicesY {
		row = showOps[posY]
		indicesX = nil
		for key := range row {
			indicesX = append(indicesX, key)
		}
		sort.Float64s(indicesX)
		for _, posX := range indicesX {
			renderBuffer.WriteString(row[posX].GetText(false))
		}
		renderBuffer.WriteString("\n")
	}
	text = renderBuffer.String()
	return
}

func InsertShowOpIntoRow(object *ShowOperation, desiredIndex float64, targetMap map[float64]*ShowOperation) (float64, error) {
	index := desiredIndex
	for f := float64(0); f <= maxRowInsertVariance; f += rowInsertionPrecision {
		if _, found := targetMap[index+f]; !found && index+f <= maxRowInsertBoundary {
			targetMap[index] = object
			return index + f, nil
		}
		if _, found := targetMap[index-f]; !found && index-f >= minRowInsertBoundary {
			targetMap[index] = object
			return index - f, nil
		}
	}
	return 0, fmt.Errorf("failed to find available space for index %f", desiredIndex)
}
