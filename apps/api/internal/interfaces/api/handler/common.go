package handler

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/schema"
)

// QueryParams 封装通用的查询参数
type QueryParams struct {
	Page            int
	PageSize        int
	Order           string
	SearchText      string
	SearchFields    []string
	QueryConditions []QueryCondition
	AllowedFields   map[string]bool
}

// QueryCondition 表示单个查询条件
type QueryCondition struct {
	Field    string
	Value    string
	Operator string
}

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 500
)

// ParsePageParams parses and bounds pagination parameters consistently across
// handlers. Invalid, zero, or negative values fall back to safe defaults.
func ParsePageParams(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), defaultPage)
	if c.Query("page") == "" {
		page = parsePositiveInt(c.Query("current"), defaultPage)
	}
	pageSize := parsePositiveInt(c.Query("pageSize"), defaultPageSize)
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

// ParseQueryParams 从请求中解析通用查询参数
func ParseQueryParams(c *gin.Context, modelType reflect.Type) QueryParams {
	searchText := c.Query("searchText")
	searchFieldsString := c.Query("searchFields")

	// 支持 page 和 current 两种页码参数名（兼容不同前端组件库）
	pageStr := c.DefaultQuery("page", "")
	if pageStr == "" {
		pageStr = c.DefaultQuery("current", "1")
	}
	page := parsePositiveInt(pageStr, defaultPage)
	pageSize := parsePositiveInt(c.DefaultQuery("pageSize", "10"), defaultPageSize)
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	order := c.DefaultQuery("order", "id desc")

	var searchFields []string
	if searchFieldsString != "" {
		searchFields = strings.Split(searchFieldsString, ",")
		for i, field := range searchFields {
			searchFields[i] = strings.TrimSpace(field)
		}
	}

	queryConditions := parseQueryConditions(c)
	allowedFields := getAllowedFields(modelType)

	return QueryParams{
		Page:            page,
		PageSize:        pageSize,
		Order:           order,
		SearchText:      searchText,
		SearchFields:    searchFields,
		QueryConditions: queryConditions,
		AllowedFields:   allowedFields,
	}
}

func parseQueryConditions(c *gin.Context) []QueryCondition {
	var conditions []QueryCondition

	queryMap := c.Request.URL.Query()

	for key, values := range queryMap {
		if key == "searchText" || key == "searchFields" || key == "page" || key == "current" || key == "pageSize" || key == "order" {
			continue
		}

		parts := strings.Split(key, "-")
		if len(parts) != 2 {
			continue
		}

		field := parts[0]
		operator := parts[1]

		if !isValidOperator(operator) {
			continue
		}

		if len(values) > 0 {
			conditions = append(conditions, QueryCondition{
				Field:    field,
				Value:    values[0],
				Operator: operator,
			})
		}
	}

	return conditions
}

func isValidOperator(operator string) bool {
	validOperators := map[string]bool{
		"like": true, "eq": true, "gt": true, "lt": true,
		"gte": true, "lte": true, "ne": true, "in": true,
	}
	return validOperators[operator]
}

// getAllowedFields derives the filter/sort allowlist from a domain entity. Fields hidden
// from JSON (`json:"-"`) and credential-like columns are never filterable or sortable,
// otherwise clients could probe values such as password hashes through query predicates.
func getAllowedFields(modelType reflect.Type) map[string]bool {
	allowedFields := make(map[string]bool)
	collectAllowedFields(modelType, allowedFields)
	return allowedFields
}

func collectAllowedFields(modelType reflect.Type, allowedFields map[string]bool) {
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if field.Anonymous {
			collectAllowedFields(field.Type, allowedFields)
			continue
		}
		if !field.IsExported() || field.Tag.Get("json") == "-" {
			continue
		}
		columnName := getColumnName(field)
		if query.IsSensitiveColumn(columnName) {
			continue
		}
		allowedFields[columnName] = true
	}
}

// ValidateOrderParam 验证排序参数，仅允许 `column [asc|desc]` 形式且列在白名单内
func ValidateOrderParam(order string, allowedFields map[string]bool) (string, error) {
	return query.NormalizeOrder(order, func(column string) bool { return allowedFields[column] })
}

// parseOrderParam reads the `order` query parameter for handlers outside GenericGets and
// validates it against the entity allowlist. It writes a 400 response when invalid.
func parseOrderParam(c *gin.Context, modelType reflect.Type, defaultOrder string) (string, bool) {
	order, err := ValidateOrderParam(c.DefaultQuery("order", defaultOrder), getAllowedFields(modelType))
	if err != nil {
		response.BadRequestI18n(c, response.ErrInvalidOrder)
		return "", false
	}
	return order, true
}

func getColumnName(field reflect.StructField) string {
	if gormTag := field.Tag.Get("gorm"); gormTag != "" {
		if strings.Contains(gormTag, "column") {
			columnParts := strings.Split(gormTag, "column:")
			if len(columnParts) > 1 {
				return strings.Split(columnParts[1], ";")[0]
			}
		}
	}

	namer := schema.NamingStrategy{}
	return namer.ColumnName("", field.Name)
}

// BuildQueryOptions 构建查询选项
func BuildQueryOptions(params QueryParams) []query.Option {
	var opts []query.Option

	if len(params.QueryConditions) > 0 {
		conditions := make([]string, 0)
		queryParams := make([]interface{}, 0)

		for _, condition := range params.QueryConditions {
			if !params.AllowedFields[condition.Field] {
				continue
			}

			switch condition.Operator {
			case "like":
				conditions = append(conditions, condition.Field+" LIKE ?")
				queryParams = append(queryParams, "%"+condition.Value+"%")
			case "eq":
				conditions = append(conditions, condition.Field+" = ?")
				queryParams = append(queryParams, condition.Value)
			case "gt":
				conditions = append(conditions, condition.Field+" > ?")
				queryParams = append(queryParams, condition.Value)
			case "lt":
				conditions = append(conditions, condition.Field+" < ?")
				queryParams = append(queryParams, condition.Value)
			case "gte":
				conditions = append(conditions, condition.Field+" >= ?")
				queryParams = append(queryParams, condition.Value)
			case "lte":
				conditions = append(conditions, condition.Field+" <= ?")
				queryParams = append(queryParams, condition.Value)
			case "ne":
				conditions = append(conditions, condition.Field+" != ?")
				queryParams = append(queryParams, condition.Value)
			case "in":
				values := strings.Split(condition.Value, ",")
				if len(values) > 0 {
					placeholders := make([]string, len(values))
					for i := range values {
						placeholders[i] = "?"
						queryParams = append(queryParams, strings.TrimSpace(values[i]))
					}
					conditions = append(conditions, condition.Field+" IN ("+strings.Join(placeholders, ",")+")")
				}
			}
		}

		if len(conditions) > 0 {
			whereClause := strings.Join(conditions, " AND ")
			opts = append(opts, *query.NewOption(whereClause, queryParams...))
		}
	}

	if params.SearchText != "" && len(params.SearchFields) > 0 {
		validFields := make([]string, 0)
		queryParams := make([]interface{}, 0)

		for _, field := range params.SearchFields {
			if params.AllowedFields[field] {
				validFields = append(validFields, field+" LIKE ?")
				queryParams = append(queryParams, "%"+params.SearchText+"%")
			}
		}

		if len(validFields) > 0 {
			whereClause := strings.Join(validFields, " OR ")
			opts = append(opts, *query.NewOption(whereClause, queryParams...))
		}
	}

	return opts
}

// GenericGets 通用的列表查询处理函数
func GenericGets(c *gin.Context, modelType reflect.Type, serviceGets func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error)) {
	params := ParseQueryParams(c, modelType)

	safeOrder, err := ValidateOrderParam(params.Order, params.AllowedFields)
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

	items, total, err := serviceGets(c, params.Page, params.PageSize, safeOrder, opts...)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessListPaged(c, total, items, params.Page, params.PageSize)
}

// GenericGet 通用的获取单个资源处理函数
func GenericGet(c *gin.Context, getFunc func(*gin.Context, uint) (interface{}, error)) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	item, err := getFunc(c, uint(id))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, item)
}

// GenericPost 通用的创建资源处理函数
func GenericPost(c *gin.Context, req interface{}, postFunc func(*gin.Context, interface{}) (interface{}, error), validateFunc func(interface{}) bool) {
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if validateFunc != nil && !validateFunc(req) {
		response.BadRequest(c)
		return
	}

	resp, err := postFunc(c, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, resp, response.InfCreateSuccess)
}

// GenericPut 通用的更新资源处理函数
func GenericPut(c *gin.Context, req interface{}, putFunc func(*gin.Context, uint, interface{}) (interface{}, error), validateFunc func(interface{}) bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if validateFunc != nil && !validateFunc(req) {
		response.BadRequest(c)
		return
	}

	item, err := putFunc(c, uint(id), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, item, response.InfUpdateSuccess)
}

// GenericPatch 通用的部分更新资源处理函数
func GenericPatch(c *gin.Context, patchFunc func(*gin.Context, uint, map[string]interface{}) (interface{}, error), validateFunc func(map[string]interface{}) bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	req := make(map[string]interface{})
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if validateFunc != nil && !validateFunc(req) {
		response.BadRequest(c)
		return
	}

	item, err := patchFunc(c, uint(id), req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, item, response.InfUpdateSuccess)
}

// GenericDelete 通用的删除资源处理函数
func GenericDelete(c *gin.Context, deleteFunc func(*gin.Context, uint) error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err = deleteFunc(c, uint(id)); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfDeleteSuccess)
}

// GenericBatchDelete 通用的批量删除资源处理函数
func GenericBatchDelete(c *gin.Context, req interface{}, batchDeleteFunc func(*gin.Context, interface{}) error, validateFunc func(interface{}) bool) {
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if validateFunc != nil && !validateFunc(req) {
		response.BadRequest(c)
		return
	}

	if err := batchDeleteFunc(c, req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfBatchDeleteSuccess)
}
