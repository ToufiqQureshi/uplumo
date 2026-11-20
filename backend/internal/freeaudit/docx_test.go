package freeaudit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// unzipDocx reads every part of a .docx and fails if the zip is broken.
func unzipDocx(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("docx is not a zip: %v", err)
	}
	parts := map[string][]byte{}
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("read %s: %v", file.Name, err)
		}
		parts[file.Name] = body
	}
	return parts
}

// requireWellFormedXML decodes the whole document; any syntax error fails.
func requireWellFormedXML(t *testing.T, name string, body []byte) {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(body))
	for {
		_, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("%s is not well-formed XML: %v", name, err)
		}
	}
}

// reportWithUntrustedText is a report whose page text has markup and an
// ampersand, which must reach the document escaped.
func reportWithUntrustedText() audit.PageReport {
	return audit.PageReport{
		URL: "https://shop.test/", CheckedAt: time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC),
		Reachable: true, Score: 59, Caps: []string{"Score capped at 59 because a critical check failed (HTTPS)."},
		Categories: []audit.CategoryScore{{ID: audit.CategoryOnPage, Name: "On-page SEO", Score: 40, Issues: 1}},
		Checks: []audit.PageCheck{{
			ID: "title", Category: audit.CategoryOnPage, Status: audit.CheckFail, Severity: "high", Weight: 5,
			Title: "Page title", Found: `"<script>alert(1)</script> & Co" (33 characters).`,
			Why: "Titles matter.", Fix: "Write a clear title & a brand.",
		}},
	}
}

func TestBuildDocxWritesAValidPackageWithEscapedPageText(t *testing.T) {
	logo := []byte("\x89PNG\r\n\x1a\nfake")
	data, err := BuildDocx(reportWithUntrustedText(), logo)
	if err != nil {
		t.Fatal(err)
	}
	parts := unzipDocx(t, data)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/_rels/document.xml.rels", "word/styles.xml", "word/media/logo.png"} {
		if _, ok := parts[name]; !ok {
			t.Errorf("part %s missing; have %v", name, keys(parts))
		}
	}
	for name, body := range parts {
		if name == "word/media/logo.png" {
			if !bytes.Equal(body, logo) {
				t.Errorf("logo bytes differ")
			}
			continue
		}
		requireWellFormedXML(t, name, body)
	}
	document := string(parts["word/document.xml"])
	if strings.Contains(document, "<script>") {
		t.Fatalf("page markup reached the document unescaped")
	}
	for _, want := range []string{"&lt;script&gt;alert(1)&lt;/script&gt; &amp; Co", "Write a clear title &amp; a brand.", "Prepared by Uplumo", "SEO audit report"} {
		if !strings.Contains(document, want) {
			t.Errorf("document does not contain %q", want)
		}
	}
	if !strings.Contains(string(parts["[Content_Types].xml"]), `Extension="png"`) {
		t.Errorf("content types do not declare png")
	}
}

func TestBuildDocxWithoutLogoOmitsTheImage(t *testing.T) {
	data, err := BuildDocx(reportWithUntrustedText(), nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := unzipDocx(t, data)
	if _, ok := parts["word/media/logo.png"]; ok {
		t.Fatalf("logo part present without a logo")
	}
	if strings.Contains(string(parts["word/_rels/document.xml.rels"]), "rIdLogo") {
		t.Fatalf("relationship to a missing logo")
	}
	requireWellFormedXML(t, "word/document.xml", parts["word/document.xml"])
}

func TestBuildDocxExplainsAnUnreachablePage(t *testing.T) {
	report := audit.PageReport{URL: "https://shop.test/", Problem: "We could not connect to this website & we tried <twice>."}
	data, err := BuildDocx(report, nil)
	if err != nil {
		t.Fatal(err)
	}
	document := string(unzipDocx(t, data)["word/document.xml"])
	if !strings.Contains(document, "We could not connect to this website &amp; we tried &lt;twice&gt;.") {
		t.Fatalf("problem text missing or unescaped")
	}
	requireWellFormedXML(t, "word/document.xml", []byte(document))
}

func TestFileHostIsSafeForAFileName(t *testing.T) {
	cases := map[string]string{
		"https://Shop.Test/page":  "shop.test",
		"https://ex\"ample.com/":  "example.com",
		"https://shop.test:8443/": "shop.test",
		"not a url %%%":           "site",
	}
	for input, want := range cases {
		if got := fileHost(input); got != want {
			t.Errorf("fileHost(%q) = %q, want %q", input, got, want)
		}
	}
}

func keys(parts map[string][]byte) []string {
	var names []string
	for name := range parts {
		names = append(names, name)
	}
	return names
}
