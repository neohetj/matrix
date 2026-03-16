package trace

import (
	"encoding/json"
	"fmt"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

type SummaryRuleMsg struct {
	id           string
	ts           int64
	msgType      string
	dataFormat   cnst.MFormat
	data         types.Data
	metadata     types.Metadata
	tracePayload *types.TracePayloadRef
}

func NewSummaryRuleMsg(msg types.RuleMsg, payloadRef *types.TracePayloadRef) *SummaryRuleMsg {
	placeholder := fmt.Sprintf(
		"[trace payload externalized: %s, %s]",
		payloadReasonLabel(payloadRef),
		formatTracePayloadSize(payloadRef.SizeBytes),
	)

	return &SummaryRuleMsg{
		id:           msg.ID(),
		ts:           msg.Ts(),
		msgType:      msg.Type(),
		dataFormat:   msg.DataFormat(),
		data:         types.Data(placeholder),
		metadata:     msg.Metadata().Copy(),
		tracePayload: cloneTracePayloadRef(payloadRef),
	}
}

func (m *SummaryRuleMsg) ID() string {
	return m.id
}

func (m *SummaryRuleMsg) Ts() int64 {
	return m.ts
}

func (m *SummaryRuleMsg) Type() string {
	return m.msgType
}

func (m *SummaryRuleMsg) DataFormat() cnst.MFormat {
	return m.dataFormat
}

func (m *SummaryRuleMsg) Data() types.Data {
	return m.data
}

func (m *SummaryRuleMsg) DataT() types.DataT {
	return nil
}

func (m *SummaryRuleMsg) Metadata() types.Metadata {
	return m.metadata
}

func (m *SummaryRuleMsg) SetData(data string, format cnst.MFormat) {
	m.data = types.Data(data)
	m.dataFormat = format
}

func (m *SummaryRuleMsg) SetMetadata(metadata types.Metadata) {
	m.metadata = metadata
}

func (m *SummaryRuleMsg) Copy() types.RuleMsg {
	return &SummaryRuleMsg{
		id:           m.id,
		ts:           m.ts,
		msgType:      m.msgType,
		dataFormat:   m.dataFormat,
		data:         m.data,
		metadata:     m.metadata.Copy(),
		tracePayload: cloneTracePayloadRef(m.tracePayload),
	}
}

func (m *SummaryRuleMsg) DeepCopy() (types.RuleMsg, error) {
	return m.Copy(), nil
}

func (m *SummaryRuleMsg) WithDataFormat(format cnst.MFormat) types.RuleMsg {
	if format.IsValid() {
		m.dataFormat = format
	} else {
		m.dataFormat = cnst.UNKNOWN
	}
	return m
}

func (m *SummaryRuleMsg) GetTracePayload() *types.TracePayloadRef {
	return cloneTracePayloadRef(m.tracePayload)
}

func (m *SummaryRuleMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id           string                 `json:"id"`
		Ts           int64                  `json:"ts"`
		MsgType      string                 `json:"msgType"`
		DataFormat   cnst.MFormat           `json:"dataFormat"`
		Data         types.Data             `json:"data"`
		DataT        types.DataT            `json:"dataT"`
		Metadata     types.Metadata         `json:"metadata"`
		TracePayload *types.TracePayloadRef `json:"tracePayload,omitempty"`
	}{
		Id:           m.id,
		Ts:           m.ts,
		MsgType:      m.msgType,
		DataFormat:   m.dataFormat,
		Data:         m.data,
		DataT:        nil,
		Metadata:     m.metadata,
		TracePayload: cloneTracePayloadRef(m.tracePayload),
	})
}

func cloneTracePayloadRef(ref *types.TracePayloadRef) *types.TracePayloadRef {
	if ref == nil {
		return nil
	}
	cloned := *ref
	return &cloned
}

func payloadReasonLabel(ref *types.TracePayloadRef) string {
	if ref == nil {
		return "payload"
	}
	switch ref.Reason {
	case "image":
		return "image payload"
	case "size":
		return "large payload"
	default:
		return "payload"
	}
}

func formatTracePayloadSize(size int) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)

	switch {
	case size >= mb:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
