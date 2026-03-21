// Package pdf generates minimal PDF files for product label printing.
// It uses no external dependencies — PDF objects are written directly.
// Supports built-in Type1 font (Helvetica) which covers Latin-1 characters.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// LabelData holds the fields to render on the label.
type LabelData struct {
	ProductName  string
	Manufacturer string
	Quantity     string
	ExpiryDate   string
	Ingredients  string
	LotNumber    string
	Country      string
	Warnings     string
	Category     string
}

// SizeMM holds the label dimensions in millimetres.
type SizeMM struct {
	Width  float64
	Height float64
}

// CommonSizes for quick reference.
var (
	Size62x29 = SizeMM{62, 29}
	Size62x100 = SizeMM{62, 100}
	SizeA4     = SizeMM{210, 297}
)

const mmToPt = 2.8346456692913

// MinFontSizeMM is the minimum mandatory font height per Reg. UE 1169/2011 art. 13.
const MinFontSizeMM = 1.2

// minFontSizePt is MinFontSizeMM converted to PDF points.
const minFontSizePt = MinFontSizeMM * mmToPt // ≈ 3.4 pt

// ValidateMinFontSize returns an error when fontSizePt is below the EU mandatory minimum.
func ValidateMinFontSize(fontSizePt float64) error {
	if fontSizePt < minFontSizePt {
		return fmt.Errorf("font size %.1f pt is below the EU mandatory minimum of %.1f pt (%.1f mm)",
			fontSizePt, minFontSizePt, MinFontSizeMM)
	}
	return nil
}

// Generate produces a minimal valid PDF rendering the label on a single page.
// Returns an error if any content font falls below the EU regulatory minimum.
func Generate(data LabelData, size SizeMM) ([]byte, error) {
	w := size.Width * mmToPt
	h := size.Height * mmToPt

	// Collect content lines
	var lines []labelLine
	if data.ProductName != "" {
		lines = append(lines, labelLine{text: safePDF(data.ProductName), size: 9, bold: true})
	}
	if data.Manufacturer != "" {
		lines = append(lines, labelLine{text: safePDF("Prod: " + data.Manufacturer), size: 7})
	}
	if data.Quantity != "" {
		lines = append(lines, labelLine{text: safePDF("Cant: " + data.Quantity), size: 7})
	}
	if data.ExpiryDate != "" {
		lines = append(lines, labelLine{text: safePDF("Exp: " + data.ExpiryDate), size: 7})
	}
	if data.LotNumber != "" {
		lines = append(lines, labelLine{text: safePDF("Lot: " + data.LotNumber), size: 7})
	}
	if data.Country != "" {
		lines = append(lines, labelLine{text: safePDF("Origine: " + data.Country), size: 7})
	}
	if data.Warnings != "" {
		lines = append(lines, labelLine{text: safePDF(data.Warnings), size: 6})
	}
	if data.Ingredients != "" {
		// Truncate long ingredient lists
		ing := data.Ingredients
		if len(ing) > 120 {
			ing = ing[:117] + "..."
		}
		lines = append(lines, labelLine{text: safePDF("Ing: " + ing), size: 5})
	}

	// Validate all font sizes meet the EU regulatory minimum.
	for _, l := range lines {
		if err := ValidateMinFontSize(float64(l.size)); err != nil {
			return nil, err
		}
	}

	// Build the content stream
	var cs bytes.Buffer
	margin := 4.0 * mmToPt
	x := margin
	y := h - margin

	cs.WriteString("BT\n")
	for _, l := range lines {
		lineH := l.size * 1.3
		y -= lineH
		if y < margin {
			break
		}
		fontName := "/F1"
		if l.bold {
			fontName = "/F2"
		}
		fmt.Fprintf(&cs, "%s %v Tf\n", fontName, l.size)
		fmt.Fprintf(&cs, "%.2f %.2f Td\n", x, y)
		fmt.Fprintf(&cs, "(%s) Tj\n", l.text)
		// Reset Td for next line (use absolute positioning next iteration)
		x = margin
		fmt.Fprintf(&cs, "%.2f %.2f Td\n", 0.0, 0.0) // no-op, position handled above
	}
	cs.WriteString("ET\n")

	// Rewrite using absolute text positioning (Tm operator)
	var cs2 bytes.Buffer
	y = h - margin
	cs2.WriteString("BT\n")
	for _, l := range lines {
		lineH := l.size * 1.3
		y -= lineH
		if y < margin {
			break
		}
		fontName := "/F1"
		if l.bold {
			fontName = "/F2"
		}
		fmt.Fprintf(&cs2, "%s %v Tf\n", fontName, l.size)
		fmt.Fprintf(&cs2, "1 0 0 1 %.2f %.2f Tm\n", margin, y)
		fmt.Fprintf(&cs2, "(%s) Tj\n", l.text)
	}
	cs2.WriteString("ET\n")
	_ = cs

	content := cs2.Bytes()

	// Assemble PDF objects
	var buf bytes.Buffer
	offsets := make([]int, 0, 8)

	writeHdr := func(s string) {
		buf.WriteString(s)
	}

	writeHdr("%PDF-1.4\n")

	obj := func(n int, body string) {
		for len(offsets) < n {
			offsets = append(offsets, 0)
		}
		offsets[n-1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}

	// 1: Catalog
	obj(1, "<< /Type /Catalog /Pages 2 0 R >>")

	// 2: Pages
	obj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")

	// 3: Page
	obj(3, fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R\n   /MediaBox [0 0 %.2f %.2f]\n   /Contents 4 0 R\n   /Resources << /Font << /F1 5 0 R /F2 6 0 R >> >> >>",
		w, h,
	))

	// 4: Content stream
	streamLen := len(content)
	obj(4, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", streamLen, content))

	// 5: Font Helvetica (regular)
	obj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")

	// 6: Font Helvetica-Bold
	obj(6, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	// Cross-reference table
	xrefOffset := buf.Len()
	numObjs := 6
	fmt.Fprintf(&buf, "xref\n0 %d\n", numObjs+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 0; i < numObjs; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}

	// Trailer
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", numObjs+1, xrefOffset)

	return buf.Bytes(), nil
}

type labelLine struct {
	text string
	size float64
	bold bool
}

// safePDF converts a string to a PDF-safe literal string (parentheses escaped,
// non-Latin-1 characters replaced with '?').
func safePDF(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			sb.WriteByte('\\')
			sb.WriteRune(r)
		case r < 128:
			sb.WriteRune(r)
		case r <= 0xff:
			// Latin-1 — keep as-is (WinAnsi covers most)
			sb.WriteRune(r)
		default:
			// Replace uncommon Unicode with ASCII equivalent or '?'
			if unicode.IsLetter(r) {
				sb.WriteByte('?')
			}
		}
	}
	return sb.String()
}
