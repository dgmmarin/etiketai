// Package zpl generates ZPL II commands for Zebra label printers.
// ZPL II reference: https://www.zebra.com/content/dam/zebra/manuals/printers/common/programming/zpl-zbi2-pm-en.pdf
package zpl

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/dgmmarin/etiketai/services/print-svc/internal/pdf"
)

// DPI constants for common Zebra printer resolutions.
const (
	DPI203 = 203
	DPI300 = 300
)

// mmToDots converts millimetres to dots at the given DPI.
func mmToDots(mm float64, dpi int) int {
	return int(mm * float64(dpi) / 25.4)
}

// Generate produces a ZPL II label string for the given label data and size.
// dpi should be 203 or 300 to match the target printer's resolution.
func Generate(data pdf.LabelData, size pdf.SizeMM, dpi int) string {
	if dpi == 0 {
		dpi = DPI203
	}

	w := mmToDots(size.Width, dpi)
	h := mmToDots(size.Height, dpi)

	var sb strings.Builder

	// Label format start
	sb.WriteString("^XA\n")

	// Label dimensions (width in dots, length in dots)
	fmt.Fprintf(&sb, "^PW%d\n", w)
	fmt.Fprintf(&sb, "^LL%d\n", h)

	// Label home (origin offset)
	sb.WriteString("^LH0,0\n")

	// Print speed + darkness
	sb.WriteString("^PR3\n") // print speed 3 ips
	sb.WriteString("^MD10\n") // media darkness 10

	// Font: use built-in Zebra font 0 (scalable)
	// ^A0N,height,width — scalable font
	margin := mmToDots(2, dpi)
	y := margin

	lineH8 := mmToDots(3, dpi)  // ~8pt line height at 203dpi
	lineH6 := mmToDots(2.2, dpi) // ~6pt line height

	// Product name — bold / large
	if data.ProductName != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH8, lineH8)
		fmt.Fprintf(&sb, "^FD%s^FS\n", safeZPL(data.ProductName))
		y += lineH8 + mmToDots(1, dpi)
	}

	// Manufacturer
	if data.Manufacturer != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		fmt.Fprintf(&sb, "^FDProd: %s^FS\n", safeZPL(data.Manufacturer))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Quantity
	if data.Quantity != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		fmt.Fprintf(&sb, "^FDCant: %s^FS\n", safeZPL(data.Quantity))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Expiry date
	if data.ExpiryDate != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		fmt.Fprintf(&sb, "^FDExp: %s^FS\n", safeZPL(data.ExpiryDate))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Lot number
	if data.LotNumber != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		fmt.Fprintf(&sb, "^FDLot: %s^FS\n", safeZPL(data.LotNumber))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Country of origin
	if data.Country != "" {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		fmt.Fprintf(&sb, "^FDOrigine: %s^FS\n", safeZPL(data.Country))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Warnings (if space)
	if data.Warnings != "" && y+lineH6 < h-margin {
		fmt.Fprintf(&sb, "^FO%d,%d\n", margin, y)
		fmt.Fprintf(&sb, "^A0N,%d,%d\n", lineH6, lineH6)
		warn := data.Warnings
		if len(warn) > 60 {
			warn = warn[:57] + "..."
		}
		fmt.Fprintf(&sb, "^FD%s^FS\n", safeZPL(warn))
		y += lineH6 + mmToDots(0.5, dpi)
	}

	// Separator line at bottom
	_ = y
	fmt.Fprintf(&sb, "^FO%d,%d^GB%d,1,1^FS\n", margin, h-margin*2, w-margin*2)

	// Print quantity (^PQ) and label format end
	sb.WriteString("^PQ1\n")
	sb.WriteString("^XZ\n")

	return sb.String()
}

// safeZPL strips characters that would break ZPL field data (^ and ~).
// Non-ASCII characters are transliterated to ASCII equivalents where possible.
func safeZPL(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '^', '~':
			// ZPL control characters — skip
		case 'ă', 'â':
			sb.WriteByte('a')
		case 'Ă', 'Â':
			sb.WriteByte('A')
		case 'î':
			sb.WriteByte('i')
		case 'Î':
			sb.WriteByte('I')
		case 'ș', 'ş':
			sb.WriteByte('s')
		case 'Ș', 'Ş':
			sb.WriteByte('S')
		case 'ț', 'ţ':
			sb.WriteByte('t')
		case 'Ț', 'Ţ':
			sb.WriteByte('T')
		default:
			if r < 128 || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || r == ' ' {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}
