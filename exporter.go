package gofpdi

import (
	"fmt"
	"github.com/tim-timpani/gofpdi/text"
	"io"
	"os"
)

const (
	PlainTextFileFormat = "plain text"
)

type Exporter struct {
	sourceFileName string
	reader         *PdfReader
}

func NewExporter(sourceFileName string) (*Exporter, error) {
	reader, err := NewPdfReader(sourceFileName)
	if err != nil {
		return nil, err
	}
	if reader.pageCount < 1 {
		return nil, fmt.Errorf("file '%s' has no pages", sourceFileName)
	}
	return &Exporter{
		sourceFileName: sourceFileName,
		reader:         reader,
	}, nil
}

// GetPagePlainText returns the plain text from a given page.  Page numbers start with 1 in the PDF world
func (e *Exporter) GetPagePlainText(pageNumber int) (string, error) {
	_, text, err := e.getTextShowOperations(pageNumber)
	return text, err
}

func (e *Exporter) ExportToPlainTextFile(fileName string) error {
	outFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	outBuffer := io.StringWriter(outFile)
	for pageNumber := 1; pageNumber <= e.reader.pageCount; pageNumber++ {
		pageText, err := e.GetPagePlainText(pageNumber)
		if err != nil {
			return err
		}
		outBuffer.WriteString(pageText)
	}
	return nil
}

func (e *Exporter) getTextShowOperations(pageNumber int) (map[int]map[float64]*text.ShowOperation, string, error) {

	// Get all the text blocks from the page contents
	blocks, err := e.reader.getPageTextBlocks(pageNumber)
	if err != nil {
		return nil, "", err
	}

	// Get all fonts used on this page
	pageFonts, err := e.reader.getFontDefinitions(pageNumber)
	if err != nil {
		return nil, "", err
	}

	// Create a page object
	page := text.NewPageRender(pageNumber, pageFonts)

	// Add the text blocks to the page object
	for _, block := range blocks {
		if err := page.AddTextBlock(block); err != nil {
			return nil, "", err
		}
	}

	return page.GetIndexedShowOps()
}
