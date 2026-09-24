package shared

// MonthlyCount 是按自然月聚合的计数。Month 形如 "2026-09"，按数据库会话时区（部署配置的
// Postgres.TimeZone）划分月份；序列连续，没有数据的月份计数为 0。
type MonthlyCount struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}
