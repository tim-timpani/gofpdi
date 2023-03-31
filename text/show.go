package text

// show.go - home of the ShowOperation which represents the "Text-Showing Operators" defined in the PDF spec. A text
// block (which is identified within the page content starting with BT token and ending with ET token) can have
// multiple text showing operators that indicate text to be rendered.  Each character (aka glyph) is precisely
// painted at a certain x/y starting position and has many characteristics that can alter its width.  Also,
// character spacing and word spacing affect how characters within the same text showing operation are subsequently
// placed.  In addition, the TJ text-showing operator can also further complicate the matter with horizontal
// adjustments that appear between individual characters or groups of characters within the same text-showing
// operand.  The text block also has operators that move the x/y position, change font, scaling, rendering, rise
// (ie superscript or subscript), knockout (visual affects).  Not all of these affect the width, but it is good
// practice to keep track of them in case we want to export to a file format that supports them.  Most fonts have
// variable width in that the individual characters are not all the same width. This makes us dependent on font
// definitions to determine placement of text.  PDFs do not have to be sequential in that it does not have to render
// text in order of x and y.  This means that it can be painting text at a certain point, then later go backwards
// or forwards to render text anywhere in relation to where it was.  Failure to pay close attention to this aspect
// of PDFs result in the final text in the wrong order or placed in odd places on the final output.  The purpose
// of the text extraction in this module is to mitigate those errors as much as possible.
//
// Here is an example of a TJ text-showing operation:
// 		[(P)1(remium )1(S)1(ubscript)0.997489(ion)]TJ
// The groups of letters are in side parentheses with horizontal adjustments between.  The total width of this
// text becomes important since it leaves the x/y position in a very particular spot that may affect other text-
// showing operations that follow within the same text block

import "strings"

type ShowOperation struct {
	chars       []*ShowChars
	PageNumber  int
	StartX      float64
	StartY      float64
	FontSize    float64
	Font        *FontDefinition
	CharSpacing float64
	WordSpacing float64
	Leading     float64
	Scale       float64
	RenderMode  int
	Rise        float64
	Knockout    float64
}

func (s *ShowOperation) AddChars(text string, adjust float64) {
	chars := ShowChars{Text: text, HorizAdjust: adjust}
	s.chars = append(s.chars, &chars)
}

func (s *ShowOperation) GetText(includeSpecial bool) string {
	output := strings.Builder{}
	for _, charsObject := range s.chars {
		for _, char := range charsObject.Text {
			if (char >= MinNonSpecialAscii && char <= MaxNonSpecialAscii) || includeSpecial {
				output.WriteByte(uint8(char))
			}
		}
	}
	return output.String()
}

func (s *ShowOperation) GetWidth() (width float64, calcErr error) {
	var glyphWidth float64
	for _, lc := range s.chars {
		for i := range lc.Text {
			glyphWidth, calcErr = s.Font.CalculateGlyphWidth(lc.Text[i], lc.HorizAdjust, s.FontSize, s.CharSpacing, s.WordSpacing, s.Scale)
			if calcErr != nil {
				return
			}
			width += glyphWidth
		}
	}
	return
}

type ShowChars struct {
	Text        string
	HorizAdjust float64
}
