package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/castorworks/castor/internal/pkg/tabular"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

const (
	// maxImportFileBytes 导入文件的体积上限；1000 行用户远用不到这么大
	maxImportFileBytes = 5 << 20
	// maxImportUnzippedBytes xlsx 解压后的体积上限，防止压缩炸弹
	maxImportUnzippedBytes = 50 << 20
)

// userImportColumn 导入文件的一列：Field 是与语言无关的字段标识，Header 是表头的 i18n key。
type userImportColumn struct {
	Field    string
	Header   string
	Required bool
}

var userImportColumns = []userImportColumn{
	{Field: dto.UserImportFieldUsername, Header: "ColumnUsername", Required: true},
	{Field: dto.UserImportFieldName, Header: "ColumnName"},
	{Field: dto.UserImportFieldEmail, Header: "ColumnEmail"},
	{Field: dto.UserImportFieldMobile, Header: "ColumnMobile"},
	{Field: dto.UserImportFieldDepartmentCode, Header: "ColumnDepartmentCode"},
}

// Export 按用户列表相同的筛选与数据范围导出
// GET /api/v1/admin/users/export
func (h *AdminUserHandler) Export(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	departments, err := h.Departments.List(c.Request.Context(), scope)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	deptNames := make(map[uint]string, len(departments))
	for _, d := range departments {
		deptNames[d.ID] = d.Name
	}
	x := newExportContext(c, h.Dictionary)
	columns := []exportColumn[dto.UserResp]{
		{"ColumnID", func(u dto.UserResp) string { return strconv.FormatUint(uint64(u.ID), 10) }},
		{"ColumnUsername", func(u dto.UserResp) string { return u.Username }},
		{"ColumnName", func(u dto.UserResp) string { return u.Name }},
		{"ColumnEmail", func(u dto.UserResp) string { return u.Email }},
		{"ColumnMobile", func(u dto.UserResp) string { return u.Mobile }},
		{"ColumnDepartment", func(u dto.UserResp) string {
			if u.DepartmentID == nil {
				return ""
			}
			return deptNames[*u.DepartmentID]
		}},
		{"ColumnAccountSource", func(u dto.UserResp) string { return x.label("account_source", u.AccountSource) }},
		{"ColumnStatus", func(u dto.UserResp) string {
			if u.Enable {
				return x.text("ValueEnabled")
			}
			return x.text("ValueDisabled")
		}},
		{"ColumnLocked", func(u dto.UserResp) string { return x.bool(u.Locked) }},
		{"ColumnCreatedAt", func(u dto.UserResp) string { return x.time(u.CreatedAt) }},
	}
	exportList(c, reflect.TypeOf(user.User{}), "users", columns, func(ctx *gin.Context, page, size int, order string, opts ...query.Option) ([]dto.UserResp, int64, error) {
		return h.UserService.Gets(ctx, scope, page, size, order, opts...)
	}, func(rows int, err error) {
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserExport, "-", fmt.Sprintf("Export %d users", rows), err)
	})
}

// ImportTemplate 下载导入模板：表头为"用户名* (username)"，括号里的字段标识与语言无关。
// GET /api/v1/admin/users/import-template
func (h *AdminUserHandler) ImportTemplate(c *gin.Context) {
	format, err := tabular.ParseFormat(c.Query("format"))
	if err != nil {
		response.HandleError(c, apperror.ErrUnsupportedFileFormat)
		return
	}
	header := make([]string, len(userImportColumns))
	for i, col := range userImportColumns {
		mark := ""
		if col.Required {
			mark = "*"
		}
		header[i] = fmt.Sprintf("%s%s (%s)", response.Message(c, col.Header), mark, col.Field)
	}
	var buf bytes.Buffer
	if err := tabular.Write(&buf, format, header, nil); err != nil {
		response.HandleError(c, err)
		return
	}
	sendTable(c, "users-import-template."+format.Extension(), format, buf.Bytes())
}

// Import 从 xlsx / CSV 批量创建用户：表单字段 file 为文件，password 为 RSA 加密的统一初始密码。
// 先整体校验，任一行有误则一个都不建，响应 data 里逐行列出问题。
// POST /api/v1/admin/users/import
func (h *AdminUserHandler) Import(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	rows, err := h.readImportFile(c)
	if err != nil {
		logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserImport, "-", "Import users", err)
		response.HandleError(c, err)
		return
	}
	result, err := h.UserService.Import(ucontext.WithAuditContext(c), scope, &dto.UserImportReq{
		Password: c.PostForm("password"),
		Rows:     rows,
	})
	created := 0
	if result != nil {
		created = result.Created
	}
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeUserImport, "-", fmt.Sprintf("Imported %d of %d users", created, len(rows)), err)
	if result != nil {
		h.publishUserCreatedNotification(c, result.CreatedIDs...)
	}
	if errors.Is(err, apperror.ErrImportInvalidRows) {
		for i := range result.Errors {
			result.Errors[i].Message = response.Message(c, result.Errors[i].Code)
		}
		response.HandleErrorWithData(c, err, result)
		return
	}
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18nWithParams(c, result, response.InfImportCompleted, map[string]any{"count": result.Created})
}

// readImportFile 读取上传的表格并按表头把每行映射到字段
func (h *AdminUserHandler) readImportFile(c *gin.Context) ([]dto.UserImportRow, error) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		return nil, apperror.ErrFileUnreadable
	}
	defer f.Close()
	if fileHeader.Size > maxImportFileBytes {
		return nil, apperror.ErrAssetTooLarge
	}
	format, err := tabular.FormatOfFile(fileHeader.Filename)
	if err != nil {
		return nil, apperror.ErrUnsupportedFileFormat
	}
	header, records, err := tabular.Read(io.LimitReader(f, maxImportFileBytes), format, tabular.Limits{
		MaxRows:          service.MaxUserImportRows,
		MaxUnzippedBytes: maxImportUnzippedBytes,
	})
	switch {
	case errors.Is(err, tabular.ErrTooManyRows):
		return nil, apperror.ErrImportTooManyRows
	case errors.Is(err, tabular.ErrEmpty):
		return nil, apperror.ErrImportEmpty
	case err != nil:
		return nil, apperror.ErrFileUnreadable
	}

	index := h.mapImportHeader(c, header)
	if _, ok := index[dto.UserImportFieldUsername]; !ok {
		return nil, apperror.ErrImportMissingColumn
	}
	if len(records) == 0 {
		return nil, apperror.ErrImportEmpty
	}
	cell := func(record tabular.Row, field string) string {
		if i, ok := index[field]; ok && i < len(record.Cells) {
			return record.Cells[i]
		}
		return ""
	}
	rows := make([]dto.UserImportRow, len(records))
	for i, record := range records {
		rows[i] = dto.UserImportRow{
			Line:           record.Line,
			Username:       cell(record, dto.UserImportFieldUsername),
			Name:           cell(record, dto.UserImportFieldName),
			Email:          cell(record, dto.UserImportFieldEmail),
			Mobile:         cell(record, dto.UserImportFieldMobile),
			DepartmentCode: cell(record, dto.UserImportFieldDepartmentCode),
		}
	}
	return rows, nil
}

// mapImportHeader 把表头单元格对应到字段：认模板里括号中的字段标识、裸字段标识，
// 或当前语言的列名（这样同语言导出的文件可以直接导回）。未识别的列忽略。
func (h *AdminUserHandler) mapImportHeader(c *gin.Context, header []string) map[string]int {
	index := map[string]int{}
	for i, raw := range header {
		name := strings.TrimSpace(raw)
		key := name
		if open := strings.LastIndex(name, "("); open >= 0 && strings.HasSuffix(name, ")") {
			key = strings.TrimSpace(name[open+1 : len(name)-1])
		}
		for _, col := range userImportColumns {
			if _, taken := index[col.Field]; taken {
				continue
			}
			if strings.EqualFold(key, col.Field) || strings.TrimSuffix(name, "*") == response.Message(c, col.Header) {
				index[col.Field] = i
				break
			}
		}
	}
	return index
}
