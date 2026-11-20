package freeaudit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// A minimal Word document (WordprocessingML in a zip) of one free audit
// report. It is built by hand so the service needs no new dependency. Every
// string from the page is untrusted and goes through xmlText.

// Page size and margins in twentieths of a point (A4, 2 cm margins).
const (
	docxPageWidth   = 11906
	docxPageHeight  = 16838
	docxMargin      = 1134
	docxTextWidth   = docxPageWidth - 2*docxMargin
	docxLogoEMU     = 548640 // 0.6 inch square
	docxMaxPriority = 10
)

// docxColumns are the widths of the check table, in twips; they sum to the
// text width.
var docxColumns = []int{1000, 2000, 2700, 2300, 1600}

// docxStatus names and colours each check status.
var docxStatus = map[audit.CheckStatus]struct{ label, color string }{
	audit.CheckFail: {"Fix", "B42318"},
	audit.CheckWarn: {"Improve", "9A6700"},
	audit.CheckPass: {"Good", "1E7B34"},
	audit.CheckInfo: {"Note", "555555"},
}

// BuildDocx renders report as a .docx file. logo is a PNG shown at the top;
// it may be nil.
func BuildDocx(report audit.PageReport, logo []byte) ([]byte, error) {
	document := &docxBody{}
	document.brand(logo)
	document.heading1(audit.KindName(report.Kind) + " audit report")
	document.paragraph(run{text: report.URL, mono: true})
	document.paragraph(run{text: "Checked " + report.CheckedAt.UTC().Format("2 January 2006, 15:04 UTC")})
	if !report.Reachable {
		document.heading2("We could not check this page")
		document.paragraph(run{text: report.Problem})
		return writeDocx(document, logo)
	}
	document.heading2(fmt.Sprintf("%s score: %d/100 (%s)", audit.KindName(report.Kind), report.Score, audit.GradeLabel(report.Score)))
	document.paragraph(run{text: audit.KindHonest(report.Kind)})
	for _, cap := range report.Caps {
		document.bullet(cap)
	}
	document.heading2("Category scores")
	document.categoryTable(report.Categories)
	document.heading2("Fix these first")
	priorities := docxPriorities(report.Checks)
	if len(priorities) == 0 {
		document.paragraph(run{text: "Nothing to fix: every check passed or is only a note."})
	}
	for _, check := range priorities {
		document.bullet(check.Title + ": " + check.Found + " " + check.Fix)
	}
	for _, category := range report.Categories {
		document.heading2(fmt.Sprintf("%s (%d/100)", category.Name, category.Score))
		document.checkTable(report.Checks, category.ID)
	}
	page := "/free-seo-audit"
	if report.Kind == audit.KindAEO {
		page = "/free-aeo-audit"
	}
	document.paragraph(run{text: "Prepared by Uplumo · https://uplumo.com — free audit at uplumo.com" + page, muted: true})
	return writeDocx(document, logo)
}

// docxPriorities returns the failed and warned checks, worst first, up to
// docxMaxPriority.
func docxPriorities(checks []audit.PageCheck) []audit.PageCheck {
	var issues []audit.PageCheck
	for _, check := range checks {
		if check.Status == audit.CheckFail || check.Status == audit.CheckWarn {
			issues = append(issues, check)
		}
	}
	slices.SortStableFunc(issues, func(a, b audit.PageCheck) int {
		if a.Status != b.Status {
			if a.Status == audit.CheckFail {
				return -1
			}
			return 1
		}
		return b.Weight - a.Weight
	})
	return issues[:min(len(issues), docxMaxPriority)]
}

// run is one piece of text with its formatting.
type run struct {
	text  string
	bold  bool
	mono  bool
	muted bool
	color string
	size  int // half-points
}

// docxBody collects the document's body XML.
type docxBody struct {
	xml bytes.Buffer
}

func (b *docxBody) brand(logo []byte) {
	if logo != nil {
		b.xml.WriteString(`<w:p><w:pPr><w:spacing w:after="60"/></w:pPr>` + inlineLogo() + `</w:p>`)
	}
	b.paragraph(run{text: "Uplumo", bold: true, size: 32, color: "0B0D0E"})
}

func (b *docxBody) heading1(text string) {
	b.xml.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` + runXML(run{text: text}) + `</w:p>`)
}

func (b *docxBody) heading2(text string) {
	b.xml.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>` + runXML(run{text: text}) + `</w:p>`)
}

func (b *docxBody) paragraph(runs ...run) {
	b.xml.WriteString("<w:p>")
	for _, r := range runs {
		b.xml.WriteString(runXML(r))
	}
	b.xml.WriteString("</w:p>")
}

// bullet writes one list item as a paragraph with a bullet character, which
// needs no numbering part.
func (b *docxBody) bullet(text string) {
	b.paragraph(run{text: "•  " + text})
}

func (b *docxBody) categoryTable(categories []audit.CategoryScore) {
	rows := [][]string{{"Category", "Score", "Passed", "To fix or improve"}}
	for _, category := range categories {
		rows = append(rows, []string{category.Name, fmt.Sprintf("%d/100", category.Score),
			fmt.Sprint(category.Passed), fmt.Sprint(category.Issues)})
	}
	b.table([]int{3600, 1800, 1800, 2400}, rows)
}

func (b *docxBody) checkTable(checks []audit.PageCheck, category string) {
	rows := [][]string{{"Status", "Check", "Found", "Why it matters", "How to fix"}}
	for _, check := range checks {
		if check.Category == category {
			rows = append(rows, []string{docxStatus[check.Status].label, check.Title, check.Found, check.Why, check.Fix})
		}
	}
	b.table(docxColumns, rows)
}

// table writes rows as a bordered table with a shaded header row. The first
// row is the header. The status column is coloured when it names a status.
func (b *docxBody) table(widths []int, rows [][]string) {
	b.xml.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="` + fmt.Sprint(sum(widths)) + `" w:type="dxa"/>` +
		`<w:tblBorders>` + borderXML("top") + borderXML("left") + borderXML("bottom") + borderXML("right") +
		borderXML("insideH") + borderXML("insideV") + `</w:tblBorders>` +
		`<w:tblLayout w:type="fixed"/></w:tblPr><w:tblGrid>`)
	for _, width := range widths {
		fmt.Fprintf(&b.xml, `<w:gridCol w:w="%d"/>`, width)
	}
	b.xml.WriteString(`</w:tblGrid>`)
	for i, row := range rows {
		b.xml.WriteString("<w:tr>")
		if i == 0 {
			b.xml.WriteString(`<w:trPr><w:tblHeader/></w:trPr>`)
		}
		for j, cell := range row {
			width := widths[min(j, len(widths)-1)]
			shade := ""
			r := run{text: cell, size: 18}
			switch {
			case i == 0:
				shade = `<w:shd w:val="clear" w:color="auto" w:fill="EDEFF2"/>`
				r.bold = true
			case j == 0:
				if status, ok := statusOf(cell); ok {
					r.bold, r.color = true, status
				}
			}
			fmt.Fprintf(&b.xml, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s</w:tcPr><w:p>%s</w:p></w:tc>`,
				width, shade, runXML(r))
		}
		b.xml.WriteString("</w:tr>")
	}
	b.xml.WriteString("</w:tbl>")
	// A paragraph between tables keeps Word from merging them.
	b.xml.WriteString(`<w:p/>`)
}

// statusOf returns the colour of a status label written in the table.
func statusOf(label string) (string, bool) {
	for _, status := range docxStatus {
		if status.label == label {
			return status.color, true
		}
	}
	return "", false
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func borderXML(side string) string {
	return fmt.Sprintf(`<w:%s w:val="single" w:sz="4" w:space="0" w:color="C9CED6"/>`, side)
}

// runXML writes one text run. Text is escaped; control characters that XML
// cannot hold are replaced.
func runXML(r run) string {
	var props strings.Builder
	if r.bold {
		props.WriteString("<w:b/>")
	}
	if r.mono {
		props.WriteString(`<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/>`)
	}
	color := r.color
	if r.muted && color == "" {
		color = "555555"
	}
	if color != "" {
		props.WriteString(`<w:color w:val="` + color + `"/>`)
	}
	if r.size > 0 {
		fmt.Fprintf(&props, `<w:sz w:val="%d"/><w:szCs w:val="%d"/>`, r.size, r.size)
	}
	rPr := ""
	if props.Len() > 0 {
		rPr = "<w:rPr>" + props.String() + "</w:rPr>"
	}
	return "<w:r>" + rPr + `<w:t xml:space="preserve">` + xmlText(r.text) + "</w:t></w:r>"
}

// xmlText escapes untrusted text for an XML text node.
func xmlText(text string) string {
	var out strings.Builder
	_ = xml.EscapeText(&out, []byte(text))
	return out.String()
}

// inlineLogo draws the logo picture (relationship rIdLogo) at docxLogoEMU.
func inlineLogo() string {
	return `<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		fmt.Sprintf(`<wp:extent cx="%d" cy="%d"/>`, docxLogoEMU, docxLogoEMU) +
		`<wp:docPr id="1" name="Uplumo logo" descr="Uplumo"/>` +
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:nvPicPr><pic:cNvPr id="1" name="logo.png"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="rIdLogo"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		fmt.Sprintf(`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>`, docxLogoEMU, docxLogoEMU) +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic>` +
		`</a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

const docxXMLHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// writeDocx packs the parts into a .docx zip.
func writeDocx(body *docxBody, logo []byte) ([]byte, error) {
	document := docxXMLHeader +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"` +
		` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"` +
		` xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"` +
		` xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"` +
		` xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>` +
		body.xml.String() +
		fmt.Sprintf(`<w:sectPr><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`,
			docxPageWidth, docxPageHeight, docxMargin, docxMargin, docxMargin, docxMargin) +
		`</w:body></w:document>`

	parts := []struct {
		name, body string
	}{
		{"[Content_Types].xml", docxContentTypes(logo != nil)},
		{"_rels/.rels", docxRootRels},
		{"word/document.xml", document},
		{"word/_rels/document.xml.rels", docxDocumentRels(logo != nil)},
		{"word/styles.xml", docxStyles},
	}
	if logo != nil {
		parts = append(parts, struct{ name, body string }{"word/media/logo.png", string(logo)})
	}
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for _, part := range parts {
		writer, err := zipWriter.Create(part.name)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", part.name, err)
		}
		if _, err := writer.Write([]byte(part.body)); err != nil {
			return nil, fmt.Errorf("write %s: %w", part.name, err)
		}
	}
	// Close writes the zip's central directory; without it the file is not a zip.
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close docx: %w", err)
	}
	return buf.Bytes(), nil
}

func docxContentTypes(withLogo bool) string {
	png := ""
	if withLogo {
		png = `<Default Extension="png" ContentType="image/png"/>`
	}
	return docxXMLHeader +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` + png +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
		`</Types>`
}

const docxRootRels = docxXMLHeader +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

func docxDocumentRels(withLogo bool) string {
	image := ""
	if withLogo {
		image = `<Relationship Id="rIdLogo" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/logo.png"/>`
	}
	return docxXMLHeader +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
		image +
		`</Relationships>`
}

// docxStyles defines Normal, Heading1 and Heading2. The fonts are the ones
// Word has on every platform.
const docxStyles = docxXMLHeader +
	`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial" w:cs="Arial"/>` +
	`<w:sz w:val="20"/><w:szCs w:val="20"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:after="120" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="240" w:after="120"/><w:outlineLvl w:val="0"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="0B0D0E"/><w:sz w:val="36"/><w:szCs w:val="36"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="240" w:after="100"/><w:outlineLvl w:val="1"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="0B0D0E"/><w:sz w:val="26"/><w:szCs w:val="26"/></w:rPr></w:style>` +
	`</w:styles>`
