package config

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// ParseByteSize 解析非负整数字节及 K/KB、M/MB、G/GB 二进制单位，不回显输入。
func ParseByteSize(raw string) (int64, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	unit := int64(1)
	for _, suffix := range []struct {
		name string
		unit int64
	}{
		{"KB", 1 << 10}, {"MB", 1 << 20}, {"GB", 1 << 30}, {"K", 1 << 10}, {"M", 1 << 20}, {"G", 1 << 30},
	} {
		if strings.HasSuffix(value, suffix.name) {
			value, unit = strings.TrimSpace(strings.TrimSuffix(value, suffix.name)), suffix.unit
			break
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 || n > math.MaxInt64/unit {
		return 0, errors.New("configuration byte_size invalid")
	}
	return n * unit, nil
}

// ParseBool 提供显式的宽松布尔语法，调用方自行决定缺失或错误的业务策略。
func ParseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "yes", "y", "on", "enabled":
		return true, nil
	case "no", "n", "off", "disabled":
		return false, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, errors.New("configuration boolean invalid")
	}
	return value, nil
}

// ParseCSV 规范化逗号列表，保留元素顺序及重复项，不扩展为业务语法解析器。
func ParseCSV(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}
