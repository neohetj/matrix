package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

// Decoder 汇集一次 typed config 装配的读取错误；调用方须在执行副作用前检查 Err。
// 它不持有全局状态，不更改来源策略，fallback 只适用于声明过的可选配置缺失。
type Decoder struct {
	ctx    context.Context
	reader types.ConfigReader
	err    error
}

// NewDecoder 为一次业务配置装配创建非并发的错误收集器。
func NewDecoder(ctx context.Context, reader types.ConfigReader) *Decoder {
	return &Decoder{ctx: ctx, reader: reader}
}

// Err 返回装配期间的全部安全错误；非 nil 时不得使用装配结果启动服务。
func (d *Decoder) Err() error { return d.err }

// decodeField 复用 Reader 类型检查，仅在正常缺失时返回调用方的业务默认值。
func decodeField[T any](d *Decoder, key string, fallback T) T {
	value, found, err := Read[T](d.ctx, d.reader, key)
	if err != nil {
		d.err = errors.Join(d.err, err)
		var zero T
		return zero
	}
	if !found {
		return fallback
	}
	return value
}

// String 读取字符串配置，Secret 是否可读由 Catalog 声明和 Reader 来源策略决定。
func (d *Decoder) String(key, fallback string) string { return decodeField(d, key, fallback) }

// Bool 读取布尔值，显式 false 不触发 fallback。
func (d *Decoder) Bool(key string, fallback bool) bool { return decodeField(d, key, fallback) }

// Int 读取机器整数，并由 Reader 检查 int64 范围。
func (d *Decoder) Int(key string, fallback int) int { return decodeField(d, key, fallback) }

// Int64 读取精确整数，显式 0 不触发 fallback。
func (d *Decoder) Int64(key string, fallback int64) int64 { return decodeField(d, key, fallback) }

// Float64 读取旧 float 类型的双精度值，不用于精确整数配置。
func (d *Decoder) Float64(key string, fallback float64) float64 { return decodeField(d, key, fallback) }

// StringList 读取 Catalog 声明的字符串数组，不在此推测分隔符。
func (d *Decoder) StringList(key string, fallback []string) []string {
	return append([]string(nil), decodeField(d, key, fallback)...)
}

// Duration 以明确单位解释裸数字，支持 Go 时长文本并传播溢出和类型错误。
func (d *Decoder) Duration(key string, bareUnit, fallback time.Duration) time.Duration {
	value, found, err := ReadDuration(d.ctx, d.reader, key, bareUnit)
	if err != nil {
		d.err = errors.Join(d.err, err)
		return 0
	}
	if !found {
		return fallback
	}
	return value
}
