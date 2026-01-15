package asset

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

// RuleMsgAsset represents a parsed rulemsg:// URI and encapsulates rulemsg handling logic.
// It also acts as a builder for rulemsg:// URIs.
type RuleMsgAsset struct {
	Scheme string
	Host   string
	Path   string
	Query  url.Values
}

// Parse parses a rulemsg:// URI into scheme, path, and query.
func ParseRuleMsg(uri string) (RuleMsgAsset, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return RuleMsgAsset{}, fmt.Errorf("invalid rulemsg uri: %w", err)
	}
	return ParseRuleMsgFromURL(u)
}

// ParseRuleMsgFromURL converts a URL object to a RuleMsgAsset struct.
func ParseRuleMsgFromURL(u *url.URL) (RuleMsgAsset, error) {
	if u.Scheme != cnst.SchemeRuleMsg {
		return RuleMsgAsset{}, fmt.Errorf("invalid rulemsg uri scheme: %s", u.Scheme)
	}
	if u.Host == "" {
		return RuleMsgAsset{}, fmt.Errorf("invalid rulemsg uri host")
	}

	return RuleMsgAsset{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   strings.TrimPrefix(u.Path, "/"),
		Query:  u.Query(),
	}, nil
}

// NormalizeAssetURI normalizes rulemsg URIs for asset collection.
func (a RuleMsgAsset) NormalizeAssetURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	if parsed.Scheme != cnst.SchemeRuleMsg || parsed.Host != cnst.DATAT {
		return uri
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	if path == "" {
		return uri
	}
	parts := strings.SplitN(path, ".", 2)
	parsed.Path = "/" + parts[0]
	return parsed.String()
}

// Validate validates rulemsg:// URIs for supported formats.
func (a RuleMsgAsset) Validate(uri *url.URL) error {
	if uri == nil {
		return fmt.Errorf("nil uri")
	}
	if uri.Scheme != cnst.SchemeRuleMsg {
		return fmt.Errorf("invalid scheme: %s", uri.Scheme)
	}
	targetScheme := uri.Host
	targetPath := strings.TrimPrefix(uri.Path, "/")

	switch targetScheme {
	case cnst.DATA:
		formatStr := uri.Query().Get("format")
		if formatStr == "" {
			return nil
		}
		format := cnst.MFormat(formatStr)
		if !format.IsValid() {
			return fmt.Errorf("invalid data format: %s", formatStr)
		}
		return nil
	case cnst.METADATA:
		return nil
	case cnst.DATAT:
		if targetPath == "" {
			return fmt.Errorf("empty dataT path")
		}
		if uri.Query().Get("sid") == "" {
			return fmt.Errorf("missing sid for dataT uri")
		}
		return nil
	default:
		return fmt.Errorf("unsupported rulemsg scheme: %s", targetScheme)
	}
}

// Handle resolves rulemsg:// URIs against the given context.
func (a RuleMsgAsset) Handle(uri *url.URL, ctx *AssetContext) (any, error) {
	msg := ctx.RuleMsg()
	if msg == nil {
		return nil, fmt.Errorf("rule message is required for rulemsg:// URI resolution")
	}

	targetScheme := uri.Host
	targetPath := strings.TrimPrefix(uri.Path, "/")

	switch targetScheme {
	case cnst.DATA:
		query := uri.Query()
		formatStr := query.Get("format")
		if formatStr == "" {
			return nil, types.InvalidParams.Wrap(fmt.Errorf("data format is required for rulemsg data reads (e.g. ?format=JSON)"))
		}
		format := cnst.MFormat(formatStr)
		if !format.IsValid() {
			return nil, types.InvalidParams.Wrap(fmt.Errorf("invalid data format: %s", formatStr))
		}
		if format != msg.DataFormat() {
			return nil, types.InvalidParams.Wrap(fmt.Errorf("rulemsg data format mismatch: expected %s, got %s", format, msg.DataFormat()))
		}

		data := string(msg.Data())
		if targetPath == "" {
			return data, nil
		}
		if msg.DataFormat() != cnst.JSON {
			return nil, types.InvalidParams.Wrap(fmt.Errorf("partial data extraction requires JSON format"))
		}
		var dataMap map[string]any
		if err := json.Unmarshal([]byte(data), &dataMap); err != nil {
			return nil, types.InternalError.Wrap(err)
		}
		val, found, err := utils.ExtractByPath(dataMap, targetPath)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, types.AssetNotFound.Wrap(fmt.Errorf("path not found in data: %s", targetPath))
		}
		return val, nil

	case cnst.METADATA:
		if targetPath == "" {
			return msg.Metadata(), nil
		}
		val, exists := msg.Metadata()[targetPath]
		if !exists {
			return nil, types.AssetNotFound.Wrap(fmt.Errorf("metadata key not found: %s", targetPath))
		}
		return val, nil

	case cnst.DATAT:
		if targetPath == "" {
			return msg.DataT(), nil
		}
		parts := strings.SplitN(targetPath, ".", 2)
		objID := parts[0]
		fieldPath := ""
		if len(parts) > 1 {
			fieldPath = parts[1]
		}

		coreObj, found := msg.DataT().Get(objID)
		if !found {
			return nil, types.AssetNotFound.Wrap(fmt.Errorf("dataT object not found: %s", objID))
		}

		if fieldPath == "" {
			return coreObj.Body(), nil
		}

		body := coreObj.Body()
		var current any
		if m, err := utils.ToMap(body); err == nil {
			current = m
		} else {
			current = body
		}
		val, found, err := utils.ExtractByPath(current, fieldPath)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, types.AssetNotFound.Wrap(fmt.Errorf("field not found in dataT object %s: %s", objID, fieldPath))
		}
		return val, nil

	default:
		return nil, types.InvalidParams.Wrap(fmt.Errorf("unsupported rulemsg scheme: %s", targetScheme))
	}
}

// Set sets a value via a rulemsg:// URI.
func (a RuleMsgAsset) Set(uri *url.URL, ctx *AssetContext, value any) error {
	msg := ctx.RuleMsg()
	if msg == nil {
		return fmt.Errorf("rule message is required for rulemsg:// URI resolution")
	}

	targetScheme := uri.Host
	targetPath := strings.TrimPrefix(uri.Path, "/")

	switch targetScheme {
	case cnst.DATA:
		if targetPath != "" {
			return types.InvalidParams.Wrap(fmt.Errorf("partial updates to rulemsg data are not allowed: %s", uri.String()))
		}
		formatStr := uri.Query().Get("format")
		format := cnst.MFormat(formatStr)
		if !format.IsValid() {
			return types.InvalidParams.Wrap(fmt.Errorf("valid data format is required for rulemsg data writes (e.g. ?format=JSON)"))
		}

		dataStr, isStr := value.(string)
		if isStr {
			if format == cnst.JSON {
				var tmp any
				if err := json.Unmarshal([]byte(dataStr), &tmp); err != nil {
					return types.InvalidParams.Wrap(fmt.Errorf("data is not valid JSON for format %s: %w", format, err))
				}
			}
		} else {
			bytes, err := json.Marshal(value)
			if err != nil {
				return types.InternalError.Wrap(fmt.Errorf("failed to marshal value for data: %w", err))
			}
			dataStr = string(bytes)
		}
		msg.SetData(dataStr, format)
		return nil

	case cnst.METADATA:
		if targetPath == "" {
			return types.InvalidParams.Wrap(fmt.Errorf("metadata key is required"))
		}
		msg.Metadata()[targetPath] = fmt.Sprintf("%v", value)
		return nil

	case cnst.DATAT:
		if targetPath == "" {
			return types.InvalidParams.Wrap(fmt.Errorf("dataT objId is required"))
		}
		parts := strings.SplitN(targetPath, ".", 2)
		objID := parts[0]
		fieldPath := ""
		if len(parts) > 1 {
			fieldPath = parts[1]
		}

		sid := uri.Query().Get("sid")
		return a.setInDataT(msg, objID, fieldPath, value, sid)

	default:
		return types.InvalidParams.Wrap(fmt.Errorf("unsupported rulemsg scheme: %s", targetScheme))
	}
}

func (a RuleMsgAsset) setInDataT(msg types.RuleMsg, objID, fieldPath string, value any, sid string) error {
	// 确保目标对象存在；若不存在则按 sid 创建。
	obj, found := msg.DataT().Get(objID)
	if !found {
		if sid == "" {
			return types.InvalidParams.Wrap(fmt.Errorf("dataT object with id '%s' not found and no defineSid provided", objID))
		}
		var err error
		obj, err = msg.DataT().NewItem(sid, objID)
		if err != nil {
			errInfo := fmt.Errorf("failed to create new dataT item with sid '%s': %w", sid, err)
			return types.InternalError.Wrap(errInfo)
		}
	}

	if fieldPath == "" {
		ok, err := utils.SetCoreObjBody(obj, value, sid)
		if err != nil {
			errInfo := fmt.Errorf("failed to set dataT object '%s' body: %w", objID, err)
			return types.InternalError.Wrap(errInfo)
		}
		if ok {
			return nil
		}
		return types.InvalidParams.Wrap(fmt.Errorf("unsupported type for whole object assignment: %T", value))
	}

	// 字段路径赋值：先转为 map，再写回强类型对象。
	objMap, err := utils.ToMap(obj.Body())
	if err != nil {
		errInfo := fmt.Errorf("failed to convert dataT object body to map for setting value: %w", err)
		return types.InternalError.Wrap(errInfo)
	}

	utils.SetValueByDotPath(objMap, fieldPath, value)

	if err := utils.Decode(objMap, obj.Body()); err != nil {
		errInfo := fmt.Errorf("failed to decode map back to dataT object body: %w", err)
		return types.InternalError.Wrap(errInfo)
	}

	return nil
}

// NewRuleMsgAsset creates a new rulemsg:// builder state.
func NewRuleMsgAsset() RuleMsgAsset {
	return RuleMsgAsset{Scheme: cnst.SchemeRuleMsg, Query: url.Values{}}
}

// Data selects rulemsg://data.
func (a RuleMsgAsset) Data() *RuleMsgAssetDataBuilder {
	a.Scheme = cnst.SchemeRuleMsg
	a.Host = cnst.DATA
	a.Path = ""
	if a.Query == nil {
		a.Query = url.Values{}
	}
	return &RuleMsgAssetDataBuilder{a: a}
}

// Metadata selects rulemsg://metadata.
func (a RuleMsgAsset) Metadata() *RuleMsgAssetMetadataBuilder {
	a.Scheme = cnst.SchemeRuleMsg
	a.Host = cnst.METADATA
	a.Path = ""
	if a.Query == nil {
		a.Query = url.Values{}
	}
	return &RuleMsgAssetMetadataBuilder{a: a}
}

// DataT selects rulemsg://dataT.
func (a RuleMsgAsset) DataT() *RuleMsgAssetDataTBuilder {
	a.Scheme = cnst.SchemeRuleMsg
	a.Host = cnst.DATAT
	a.Path = ""
	if a.Query == nil {
		a.Query = url.Values{}
	}
	return &RuleMsgAssetDataTBuilder{a: a}
}

// Build assembles the URI string.
func (a RuleMsgAsset) Build() string {
	scheme := a.Scheme
	if scheme == "" {
		scheme = cnst.SchemeRuleMsg
	}
	if a.Query == nil {
		a.Query = url.Values{}
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     a.Host,
		Path:     a.Path,
		RawQuery: a.Query.Encode(),
	}
	return u.String()
}

// RuleMsgAssetDataBuilder builds rulemsg://data URIs.
type RuleMsgAssetDataBuilder struct {
	a RuleMsgAsset
}

// Path sets the data path (e.g., "key" or "nested.key").
func (d *RuleMsgAssetDataBuilder) Path(path string) *RuleMsgAssetDataBuilder {
	if path == "" {
		d.a.Path = ""
		return d
	}
	d.a.Path = "/" + strings.TrimPrefix(path, "/")
	return d
}

// Format sets the data format query (e.g., JSON).
func (d *RuleMsgAssetDataBuilder) Format(format cnst.MFormat) *RuleMsgAssetDataBuilder {
	if format.IsValid() {
		if d.a.Query == nil {
			d.a.Query = url.Values{}
		}
		d.a.Query.Set("format", string(format))
	}
	return d
}

// Build assembles the URI string.
func (d *RuleMsgAssetDataBuilder) Build() string {
	return d.a.Build()
}

// RuleMsgAssetMetadataBuilder builds rulemsg://metadata URIs.
type RuleMsgAssetMetadataBuilder struct {
	a RuleMsgAsset
}

// Key sets the metadata key.
func (m *RuleMsgAssetMetadataBuilder) Key(key string) *RuleMsgAssetMetadataBuilder {
	if key == "" {
		m.a.Path = ""
		return m
	}
	m.a.Path = "/" + strings.TrimPrefix(key, "/")
	return m
}

// Build assembles the URI string.
func (m *RuleMsgAssetMetadataBuilder) Build() string {
	return m.a.Build()
}

// RuleMsgAssetDataTBuilder builds rulemsg://dataT URIs.
type RuleMsgAssetDataTBuilder struct {
	a     RuleMsgAsset
	objID string
	field string
}

// Obj sets the DataT object ID.
func (d *RuleMsgAssetDataTBuilder) Obj(objID string) *RuleMsgAssetDataTBuilder {
	d.objID = objID
	return d
}

// Field sets the DataT field path.
func (d *RuleMsgAssetDataTBuilder) Field(field string) *RuleMsgAssetDataTBuilder {
	d.field = field
	return d
}

// SID sets the DataT SID query parameter.
func (d *RuleMsgAssetDataTBuilder) SID(sid string) *RuleMsgAssetDataTBuilder {
	if sid != "" {
		if d.a.Query == nil {
			d.a.Query = url.Values{}
		}
		d.a.Query.Set("sid", sid)
	}
	return d
}

// Build assembles the URI string.
func (d *RuleMsgAssetDataTBuilder) Build() string {
	path := ""
	if d.objID != "" {
		path = d.objID
		if d.field != "" {
			path = path + "." + d.field
		}
	}
	if path != "" {
		d.a.Path = "/" + strings.TrimPrefix(path, "/")
	}
	return d.a.Build()
}

// DataURI creates a URI for accessing RuleMsg.Data with explicit format.
func (a RuleMsgAsset) DataURI(format cnst.MFormat) string {
	return NewRuleMsgAsset().Data().Format(format).Build()
}

// MetadataURI creates a URI for accessing RuleMsg.Metadata.
func (a RuleMsgAsset) MetadataURI(key string) string {
	return NewRuleMsgAsset().Metadata().Key(key).Build()
}

// DataTURI creates a URI for accessing DataT with explicit SID.
func (a RuleMsgAsset) DataTURI(objID string, fieldPath string, sid string) string {
	return NewRuleMsgAsset().DataT().Obj(objID).Field(fieldPath).SID(sid).Build()
}
