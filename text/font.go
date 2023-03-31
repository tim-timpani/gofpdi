package text

import "fmt"

type FontDefinition struct {
	Name       string
	Base       string
	Type       int
	FirstChar  uint8
	LastChar   uint8
	Widths     []int
	Descriptor int
}

// CalculateGlyphWidth - for horizontal writing, this calculation (based on PDF documentation) is how the x/y position
// is determined after the glyph is painted.
func (f *FontDefinition) CalculateGlyphWidth(char uint8, tjAdjustment float64, fontSize float64, charSpacing float64,
	wordSpacing float64, horizontalScaling float64) (width float64, calcErr error) {
	if char < f.FirstChar || char > f.LastChar {
		calcErr = fmt.Errorf("char %d is outside range of allowed values for font %s", char, f.Name)
		return
	}
	width = (float64(f.Widths[char-f.FirstChar]) - tjAdjustment/1000) * fontSize
	if char == 32 {
		width += wordSpacing
	} else {
		width += charSpacing
	}
	// TODO: vertical writing would not add any horizontal scaling per the PDF spec.  Add another arg if we need to support it
	width = width * horizontalScaling
	return
}
