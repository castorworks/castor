// Package tabular 读写导入导出用的表格文件（CSV 与 Excel .xlsx）。
//
// 只处理"一张表头 + 若干行字符串"这一种形状：列的含义、取值与校验归业务模块。
// 写出时 CSV 带 UTF-8 BOM（Excel 才能正确识别中文），并对以公式字符开头的单元格
// 加前缀，避免 CSV 注入；xlsx 的单元格一律写成字符串，不会被当作公式求值。
// 读取时限制文件体积、解压体积与行数，只读第一张工作表。
package tabular

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// Format 表格文件格式
type Format string

const (
	CSV  Format = "csv"
	XLSX Format = "xlsx"
)

var (
	// ErrUnsupportedFormat 既不是 CSV 也不是 xlsx
	ErrUnsupportedFormat = errors.New("tabular: unsupported file format")
	// ErrUnreadable 文件损坏、不是声称的格式，或 CSV 不是 UTF-8 编码
	ErrUnreadable = errors.New("tabular: file cannot be read")
	// ErrTooManyRows 数据行超过上限
	ErrTooManyRows = errors.New("tabular: too many rows")
	// ErrEmpty 没有表头
	ErrEmpty = errors.New("tabular: file is empty")
)

// ParseFormat 解析导出参数里的格式；空串按 xlsx 处理。
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(XLSX):
		return XLSX, nil
	case string(CSV):
		return CSV, nil
	}
	return "", ErrUnsupportedFormat
}

// FormatOfFile 按上传文件名的扩展名判断格式。
func FormatOfFile(name string) (Format, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		return CSV, nil
	case ".xlsx":
		return XLSX, nil
	}
	return "", ErrUnsupportedFormat
}

// ContentType 下载响应的 Content-Type
func (f Format) ContentType() string {
	if f == CSV {
		return "text/csv; charset=utf-8"
	}
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// Extension 文件扩展名（不含点号）
func (f Format) Extension() string { return string(f) }

// sheetName 导出文件唯一工作表的名称
const sheetName = "Sheet1"

// formulaPrefixes 以这些字符开头的单元格会被电子表格当作公式
const formulaPrefixes = "=+-@\t\r"

// neutralize 给可能被当作公式的 CSV 单元格加单引号前缀（OWASP CSV Injection）。
func neutralize(cell string) string {
	if cell != "" && strings.ContainsRune(formulaPrefixes, rune(cell[0])) {
		return "'" + cell
	}
	return cell
}

// restore 撤销 neutralize：导出的文件原样导回时得到原值。
func restore(cell string) string {
	if len(cell) >= 2 && cell[0] == '\'' && strings.ContainsRune(formulaPrefixes, rune(cell[1])) {
		return cell[1:]
	}
	return cell
}

// Write 把表头与数据行写成 format 格式。
func Write(w io.Writer, format Format, header []string, rows [][]string) error {
	switch format {
	case CSV:
		return writeCSV(w, header, rows)
	case XLSX:
		return writeXLSX(w, header, rows)
	}
	return ErrUnsupportedFormat
}

func writeCSV(w io.Writer, header []string, rows [][]string) error {
	if _, err := w.Write([]byte("\ufeff")); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	// 统一用 CRLF：Windows 上的 Excel 与记事本都按行显示。
	cw.UseCRLF = true
	record := make([]string, len(header))
	for i, cell := range header {
		record[i] = neutralize(cell)
	}
	if err := cw.Write(record); err != nil {
		return err
	}
	for _, row := range rows {
		record = record[:0]
		for _, cell := range row {
			record = append(record, neutralize(cell))
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeXLSX(w io.Writer, header []string, rows [][]string) error {
	file := excelize.NewFile()
	defer file.Close()
	stream, err := file.NewStreamWriter(sheetName)
	if err != nil {
		return err
	}
	bold, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	// 按表头与前若干行估一个列宽，免得打开后全是 ####。
	for col := range header {
		width := displayWidth(header[col])
		for i := 0; i < len(rows) && i < 200; i++ {
			if col < len(rows[i]) {
				width = max(width, displayWidth(rows[i][col]))
			}
		}
		if err := stream.SetColWidth(col+1, col+1, float64(min(max(width, 8), 60))+2); err != nil {
			return err
		}
	}
	if err := stream.SetRow("A1", cellsOf(header, bold)); err != nil {
		return err
	}
	for i, row := range rows {
		cellRef, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		if err := stream.SetRow(cellRef, cellsOf(row, 0)); err != nil {
			return err
		}
	}
	if err := stream.Flush(); err != nil {
		return err
	}
	return file.Write(w)
}

// cellsOf 把每个单元格都写成字符串：以 = 开头的内容也不会成为公式。
func cellsOf(values []string, style int) []any {
	cells := make([]any, len(values))
	for i, v := range values {
		cells[i] = excelize.Cell{StyleID: style, Value: v}
	}
	return cells
}

// displayWidth 粗略的显示宽度：全角字符按 2 计。
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r > 0x2E80 {
			width += 2
		} else {
			width++
		}
	}
	return width
}

// Row 一行数据。Line 是它在文件里的行号（表头为第 1 行），用于报错定位。
type Row struct {
	Line  int
	Cells []string
}

// Limits 读取上限
type Limits struct {
	// MaxRows 数据行（不含表头与空行）上限
	MaxRows int
	// MaxUnzippedBytes xlsx 解压后的总大小上限，防止压缩炸弹
	MaxUnzippedBytes int64
}

// Read 读取第一张工作表：返回表头与非空数据行，单元格已去掉首尾空白。
// 调用方负责限制 r 的总字节数（上传体积上限）。
func Read(r io.Reader, format Format, limits Limits) ([]string, []Row, error) {
	var (
		records []Row
		err     error
	)
	switch format {
	case CSV:
		records, err = readCSV(r)
	case XLSX:
		records, err = readXLSX(r, limits)
	default:
		return nil, nil, ErrUnsupportedFormat
	}
	if err != nil {
		return nil, nil, err
	}

	headerAt := -1
	for i, record := range records {
		if !blank(record.Cells) {
			headerAt = i
			break
		}
	}
	if headerAt < 0 {
		return nil, nil, ErrEmpty
	}
	// 只有 CSV 导出时加过单引号前缀；xlsx 单元格本就是字符串，原样读取。
	undo := format == CSV
	header := clean(records[headerAt].Cells, undo)
	var rows []Row
	for _, record := range records[headerAt+1:] {
		if blank(record.Cells) {
			continue
		}
		if len(rows) == limits.MaxRows {
			return nil, nil, ErrTooManyRows
		}
		rows = append(rows, Row{Line: record.Line, Cells: clean(record.Cells, undo)})
	}
	return header, rows, nil
}

func readCSV(r io.Reader) ([]Row, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	// 只接受 UTF-8：Excel 默认另存的 GBK/Shift_JIS CSV 在这里明确报错，
	// 而不是悄悄导入一堆乱码。
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%w: CSV is not UTF-8 encoded", ErrUnreadable)
	}
	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	// csv.Reader 跳过空行，行号取自记录本身的位置。
	var records []Row
	for {
		cells, err := cr.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
		}
		line, _ := cr.FieldPos(0)
		records = append(records, Row{Line: line, Cells: cells})
	}
}

func readXLSX(r io.Reader, limits Limits) (records []Row, err error) {
	// 构造出的畸形文件能让 excelize 读取时 panic（GO-2026-6452，上游尚无修复版本）：
	// 上传的文件不可信，panic 一律当作无法读取处理。
	defer func() {
		if recovered := recover(); recovered != nil {
			records, err = nil, fmt.Errorf("%w: %v", ErrUnreadable, recovered)
		}
	}()
	opts := excelize.Options{}
	if limits.MaxUnzippedBytes > 0 {
		opts.UnzipSizeLimit = limits.MaxUnzippedBytes
		opts.UnzipXMLSizeLimit = limits.MaxUnzippedBytes
	}
	file, err := excelize.OpenReader(r, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	defer file.Close()
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrEmpty
	}
	cells, err := getRows(file, sheets[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	records = make([]Row, len(cells))
	for i := range cells {
		records[i] = Row{Line: i + 1, Cells: cells[i]}
	}
	return records, nil
}

// getRows 读取工作表的全部行；测试替换它来模拟 excelize 的 panic。
var getRows = func(file *excelize.File, sheet string) ([][]string, error) {
	return file.GetRows(sheet)
}

func blank(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func clean(record []string, undoNeutralize bool) []string {
	out := make([]string, len(record))
	for i, cell := range record {
		out[i] = strings.TrimSpace(cell)
		if undoNeutralize {
			out[i] = restore(out[i])
		}
	}
	return out
}
