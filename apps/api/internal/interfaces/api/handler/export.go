package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/tabular"
	"github.com/gin-gonic/gin"
)

const (
	// maxExportRows 单次导出的行数上限；超过时要求缩小筛选范围，而不是悄悄截断
	maxExportRows = 10000
	// exportPageSize 导出时逐页读取的页大小
	exportPageSize = 500
	// exportTimeLayout 导出文件里的时间格式
	exportTimeLayout = "2006-01-02 15:04:05"
)

// exportColumn 导出文件的一列：Header 是表头的 i18n key，Value 取单元格文本。
type exportColumn[T any] struct {
	Header string
	Value  func(T) string
}

// exportContext 渲染单元格所需的请求上下文：语言、时区与字典标签。
type exportContext struct {
	c        *gin.Context
	lang     string
	location *time.Location
	dict     service.DictionaryService
	labels   map[string]map[string]string
}

func newExportContext(c *gin.Context, dict service.DictionaryService) *exportContext {
	location := time.Local
	// tz 是浏览器所在时区（IANA 名称）；无效时退回 API 进程的时区。
	if tz := c.Query("tz"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			location = loc
		}
	}
	return &exportContext{c: c, lang: response.Language(c), location: location, dict: dict, labels: map[string]map[string]string{}}
}

// time 按请求时区格式化时间；零值为空。
func (x *exportContext) time(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(x.location).Format(exportTimeLayout)
}

// text 返回 i18n key 在请求语言下的文案
func (x *exportContext) text(key string) string {
	return response.Message(x.c, key)
}

func (x *exportContext) bool(v bool) string {
	if v {
		return x.text("ValueYes")
	}
	return x.text("ValueNo")
}

// label 返回字典值在请求语言下的标签；字典缺失该值时原样输出。
func (x *exportContext) label(typeCode, value string) string {
	labels, ok := x.labels[typeCode]
	if !ok {
		labels = map[string]string{}
		if items, err := x.dict.GetItemsByTypeCode(x.c.Request.Context(), typeCode); err == nil {
			for _, item := range items {
				labels[item.Value] = item.Label.Localized(x.lang)
			}
		}
		x.labels[typeCode] = labels
	}
	if label, ok := labels[value]; ok && label != "" {
		return label
	}
	return value
}

// exportList 按列表接口相同的筛选、搜索与排序（以及 fetch 内置的数据范围）逐页读出全部匹配行，
// 写成 format 参数指定的文件下发。参数校验通过后（真正尝试导出时）无论成败都调用一次 audit，
// 超出行数上限、查询失败也要留痕；出错时已写好错误响应。
func exportList[T any](c *gin.Context, modelType reflect.Type, name string, columns []exportColumn[T],
	fetch func(ctx *gin.Context, page, size int, order string, opts ...query.Option) ([]T, int64, error),
	audit func(rows int, err error)) {
	format, err := tabular.ParseFormat(c.Query("format"))
	if err != nil {
		response.HandleError(c, apperror.ErrUnsupportedFileFormat)
		return
	}
	params := ParseQueryParams(c, modelType)
	order, err := ValidateOrderParam(params.Order, params.AllowedFields)
	if err != nil {
		response.BadRequestI18n(c, response.ErrInvalidOrder)
		return
	}
	for _, field := range params.SearchFields {
		if !params.AllowedFields[field] {
			response.BadRequestI18n(c, response.ErrInvalidSearchField)
			return
		}
	}
	opts := BuildQueryOptions(params)

	data, n, err := buildExport(c, format, columns, func(page int) ([]T, int64, error) {
		return fetch(c, page, exportPageSize, order, opts...)
	})
	audit(n, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	sendTable(c, fmt.Sprintf("%s-%s.%s", name, time.Now().Format("20060102-150405"), format.Extension()), format, data)
}

// buildExport 逐页读出全部匹配行并写成表格；超过 maxExportRows 时拒绝而不是截断。
func buildExport[T any](c *gin.Context, format tabular.Format, columns []exportColumn[T], fetch func(page int) ([]T, int64, error)) ([]byte, int, error) {
	var items []T
	for page := 1; ; page++ {
		batch, total, err := fetch(page)
		if err != nil {
			return nil, 0, err
		}
		if total > maxExportRows {
			return nil, 0, apperror.ErrExportTooLarge
		}
		items = append(items, batch...)
		if len(batch) < exportPageSize || int64(len(items)) >= total {
			break
		}
	}

	header := make([]string, len(columns))
	for i, col := range columns {
		header[i] = response.Message(c, col.Header)
	}
	rows := make([][]string, len(items))
	for i, item := range items {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = col.Value(item)
		}
		rows[i] = row
	}
	var buf bytes.Buffer
	if err := tabular.Write(&buf, format, header, rows); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(items), nil
}

// sendTable 以附件形式下发生成的表格文件
func sendTable(c *gin.Context, filename string, format tabular.Format, data []byte) {
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, format.ContentType(), data)
}

func (x *exportContext) result(success bool) string {
	if success {
		return x.text("ValueSucceeded")
	}
	return x.text("ValueFailed")
}
