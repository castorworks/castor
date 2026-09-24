package tabular

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

var (
	header = []string{"用户名", "Email", "Mobile"}
	rows   = [][]string{
		{"alice", "alice@example.com", "+8613800000000"},
		{"=HYPERLINK(\"http://evil\")", "@SUM(1)", "-1"},
		{"中文名字", "", ""},
	}
)

func TestFormatParsing(t *testing.T) {
	for in, want := range map[string]Format{"": XLSX, "xlsx": XLSX, " CSV ": CSV} {
		if got, err := ParseFormat(in); err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseFormat("xls"); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("ParseFormat(xls) err = %v", err)
	}
	for name, want := range map[string]Format{"a.CSV": CSV, "dir/b.xlsx": XLSX} {
		if got, err := FormatOfFile(name); err != nil || got != want {
			t.Errorf("FormatOfFile(%q) = %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"a.xls", "a.txt", "noext"} {
		if _, err := FormatOfFile(name); !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("FormatOfFile(%q) should be unsupported", name)
		}
	}
}

func TestCSVNeutralizesFormulasAndRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, CSV, header, rows); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\ufeff") {
		t.Fatal("CSV must start with a UTF-8 BOM so Excel detects the encoding")
	}
	for _, dangerous := range []string{"\r\n=HYPERLINK", ",@SUM", ",-1"} {
		if strings.Contains(out, dangerous) {
			t.Fatalf("formula-like cell written as is (%q):\n%s", dangerous, out)
		}
	}
	if !strings.Contains(out, "'+8613800000000") || !strings.Contains(out, `"'=HYPERLINK(""http://evil"")"`) {
		t.Fatalf("formula-like cells should carry a quote prefix:\n%s", out)
	}

	gotHeader, gotRows, err := Read(bytes.NewReader(buf.Bytes()), CSV, Limits{MaxRows: 10})
	if err != nil {
		t.Fatal(err)
	}
	assertTable(t, gotHeader, gotRows)
}

func TestXLSXWritesStringsAndRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, XLSX, header, rows); err != nil {
		t.Fatal(err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if formula, _ := file.GetCellFormula(sheetName, "A3"); formula != "" {
		t.Fatalf("cell A3 became a formula: %q", formula)
	}
	if v, _ := file.GetCellValue(sheetName, "A3"); v != rows[1][0] {
		t.Fatalf("A3 = %q", v)
	}

	gotHeader, gotRows, err := Read(bytes.NewReader(buf.Bytes()), XLSX, Limits{MaxRows: 10, MaxUnzippedBytes: 10 << 20})
	if err != nil {
		t.Fatal(err)
	}
	assertTable(t, gotHeader, gotRows)
}

func assertTable(t *testing.T, gotHeader []string, gotRows []Row) {
	t.Helper()
	if strings.Join(gotHeader, "|") != strings.Join(header, "|") {
		t.Fatalf("header = %q", gotHeader)
	}
	if len(gotRows) != len(rows) {
		t.Fatalf("rows = %d, want %d", len(gotRows), len(rows))
	}
	for i, row := range gotRows {
		if row.Line != i+2 {
			t.Errorf("row %d line = %d", i, row.Line)
		}
		for j, want := range rows[i] {
			got := ""
			if j < len(row.Cells) {
				got = row.Cells[j]
			}
			if got != want {
				t.Errorf("row %d col %d = %q, want %q", i, j, got, want)
			}
		}
	}
}

func TestReadSkipsBlankLinesAndKeepsLineNumbers(t *testing.T) {
	data := "\n username , name \n\n bob , Bob \n , \n carol,Carol\n"
	h, rs, err := Read(strings.NewReader(data), CSV, Limits{MaxRows: 10})
	if err != nil {
		t.Fatal(err)
	}
	if h[0] != "username" || h[1] != "name" || len(rs) != 2 || rs[0].Line != 4 || rs[1].Line != 6 || rs[1].Cells[0] != "carol" {
		t.Fatalf("header %q rows %+v", h, rs)
	}
}

func TestReadLimitsAndErrors(t *testing.T) {
	if _, _, err := Read(strings.NewReader("a\n1\n2\n3\n"), CSV, Limits{MaxRows: 2}); !errors.Is(err, ErrTooManyRows) {
		t.Errorf("too many rows err = %v", err)
	}
	if _, _, err := Read(strings.NewReader("\n , \n"), CSV, Limits{MaxRows: 2}); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty err = %v", err)
	}
	gbk := []byte{0xd3, 0xc3, 0xbb, 0xa7, 0xc3, 0xfb, '\n'} // "用户名" in GBK
	if _, _, err := Read(bytes.NewReader(gbk), CSV, Limits{MaxRows: 2}); !errors.Is(err, ErrUnreadable) {
		t.Errorf("non UTF-8 err = %v", err)
	}
	if _, _, err := Read(strings.NewReader("not a zip"), XLSX, Limits{MaxRows: 2}); !errors.Is(err, ErrUnreadable) {
		t.Errorf("broken xlsx err = %v", err)
	}
}

func TestReadRejectsXLSXOverUnzipLimit(t *testing.T) {
	var buf bytes.Buffer
	big := make([][]string, 2000)
	for i := range big {
		big[i] = []string{strings.Repeat("x", 200)}
	}
	if err := Write(&buf, XLSX, []string{"h"}, big); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(bytes.NewReader(buf.Bytes()), XLSX, Limits{MaxRows: 5000, MaxUnzippedBytes: 64 << 10}); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("xlsx over the unzip limit err = %v; want ErrUnreadable", err)
	}
}

// 读取时 excelize 的 panic（GO-2026-6452：构造的共享字符串下标可触发）按无法读取处理，不会扩散。
func TestReadTurnsXLSXPanicsIntoUnreadable(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, XLSX, []string{"username"}, [][]string{{"alice"}}); err != nil {
		t.Fatal(err)
	}
	original := getRows
	t.Cleanup(func() { getRows = original })
	getRows = func(*excelize.File, string) ([][]string, error) { panic("runtime error: index out of range [-1]") }
	if _, _, err := Read(bytes.NewReader(buf.Bytes()), XLSX, Limits{MaxRows: 10}); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, want ErrUnreadable", err)
	}
}

// 负的共享字符串下标这类畸形内容：读取要么报无法读取，要么当作空白，不能出错崩溃。
func TestReadSurvivesMalformedXLSX(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="1" uniqueCount="1"><si><t>username</t></si></sst>`,
		"xl/worksheets/sheet1.xml":   `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>-1</v></c></row><row r="2"><c r="A2" t="s"><v>-5</v></c></row></sheetData></worksheet>`,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	// 不 panic 即可：上游可能报错，也可能返回空白单元格。
	_, _, err := Read(bytes.NewReader(buf.Bytes()), XLSX, Limits{MaxRows: 10, MaxUnzippedBytes: 1 << 20})
	if err != nil && !errors.Is(err, ErrUnreadable) && !errors.Is(err, ErrEmpty) {
		t.Fatalf("malformed xlsx err = %v", err)
	}
}
