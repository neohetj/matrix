package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// McpArgumentPolicy 声明模块禁止调用方提交的字段；规则只能增加通用保护。
// endpoint 必须显式声明该对象，空对象表示只使用通用及 authContexts Header 保护。
type McpArgumentPolicy struct {
	DenyKeys     []string `json:"denyKeys,omitempty"`
	DenyPrefixes []string `json:"denyPrefixes,omitempty"`
}

// UnmarshalJSON 拒绝策略内未知字段，避免安全配置拼写错误被静默忽略。
func (p *McpArgumentPolicy) UnmarshalJSON(data []byte) error {
	type policy McpArgumentPolicy
	var decoded policy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return fmt.Errorf("mcp argumentPolicy: %w", err)
	}
	*p = McpArgumentPolicy(decoded)
	return nil
}
