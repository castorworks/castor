package shared

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

type CustomTime time.Time

func (t *CustomTime) UnmarshalJSON(data []byte) (err error) {
	if len(data) == 2 {
		*t = CustomTime(time.Time{})
		return
	}

	now, err := time.Parse(`"`+TimeFormat+`"`, string(data))
	*t = CustomTime(now)
	return
}

func (t CustomTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(TimeFormat)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, TimeFormat)
	b = append(b, '"')
	return b, nil
}

func (t CustomTime) Value() (driver.Value, error) {
	if t.String() == "0001-01-01 00:00:00" {
		return nil, nil
	}
	return []byte(time.Time(t).Format(TimeFormat)), nil
}

// Scan 实现 sql.Scanner；与 Value 对称，字符串按本地时区的 TimeFormat 解析
func (t *CustomTime) Scan(v interface{}) error {
	switch value := v.(type) {
	case nil:
		*t = CustomTime(time.Time{})
		return nil
	case time.Time:
		*t = CustomTime(value)
		return nil
	case []byte:
		return t.parse(string(value))
	case string:
		return t.parse(value)
	default:
		return fmt.Errorf("shared: cannot scan %T into CustomTime", v)
	}
}

func (t *CustomTime) parse(value string) error {
	parsed, err := time.ParseInLocation(TimeFormat, value, time.Local)
	if err != nil {
		return fmt.Errorf("shared: parse CustomTime %q: %w", value, err)
	}
	*t = CustomTime(parsed)
	return nil
}

func (t CustomTime) String() string {
	return time.Time(t).Format(TimeFormat)
}
