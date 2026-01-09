package cnst

const (
	SchemeRuleMsg = "rulemsg"
	SchemeConfig  = "config"
	SchemeRel     = "rel"
	SchemeRef     = "ref"
	Prefix        = "://"
	// RefPrefix is the prefix for referencing a shared node instance from the node pool.
	RefPrefix = SchemeRef + Prefix
	// RuleMsgPrefix is the prefix for accessing RuleMsg data.
	RuleMsgPrefix = SchemeRuleMsg + Prefix
	// ConfigPrefix is the prefix for accessing configuration data.
	ConfigPrefix = SchemeConfig + Prefix
	// RelPrefix is the prefix for accessing relative file resources.
	RelPrefix = SchemeRel + Prefix

	// ViewTypeStaticTopology represents a static, non-executable graph showing deployment or logical relationships.
	// It primarily uses the 'relations' field for visualization.
	ViewTypeStaticTopology ViewType = "static-topology"

	// ViewTypeExecutionFlow represents an executable DAG (Directed Acyclic Graph) that defines a workflow.
	// It primarily uses the 'connections' field for visualization.
	ViewTypeExecutionFlow ViewType = "execution-flow"

	// ViewTypeHybrid represents a view that combines both logical relations and execution connections.
	ViewTypeHybrid ViewType = "hybrid"
)

const (
	DATA     string = "data"
	DATAT    string = "dataT"
	METADATA string = "metadata"
)

const (
	// Basic Type SIDs
	SID_STRING               = "String"
	SID_INT64                = "Int64"
	SID_FLOAT64              = "Float64"
	SID_BOOL                 = "Bool"
	SID_MAP_STRING_STRING    = "MapStringString"
	SID_MAP_STRING_INTERFACE = "MapStringInterface"
)

const LIST_PREFIX = "[]"

const (
	STRING MType = "string"
	INT    MType = "int"
	INT64  MType = "int64"
	FLOAT  MType = "float"
	BOOL   MType = "bool"
	OBJECT MType = "object"
	MAP    MType = "map"
	FILE   MType = "file"
	ARRAY  MType = "array"
	ANY    MType = "any"
)

func (m MType) IsSupported() bool {
	switch m {
	case STRING, INT, INT64, FLOAT, BOOL, OBJECT, MAP, FILE, ARRAY, ANY:
		return true
	default:
		// Check for array types (e.g. "[]string")
		if isList, _ := m.IsList(); isList {
			return true
		}
		return false
	}
}

const (
	JSON    MFormat = "JSON"
	TEXT    MFormat = "TEXT"
	BYTES   MFormat = "BYTES"
	IMAGE   MFormat = "IMAGE"
	UNKNOWN MFormat = "UNKNOWN"
)

func (m MFormat) IsValid() bool {
	switch m {
	case JSON, TEXT, BYTES, IMAGE:
		return true
	default:
		return false
	}
}

// Predefined error codes for the Matrix engine.
// The code format is aabbbcccc, where:
// aa: 20 (Software Product Department)
// bbb: Module (000: Global, 001: Runtime, 002: Parser, etc.)
// cccc: Specific error identifier.
const (
	// Global Errors (20000xxxx)
	CodeInternalError        ErrCode = "200000001"
	CodeInvalidParams        ErrCode = "200000002"
	CodeInvalidConfiguration ErrCode = "200000003"

	// Runtime Errors (20001xxxx)
	CodeNodeNotFound  ErrCode = "200010001"
	CodeFuncNotFound  ErrCode = "200010002"
	CodeNodePoolNil   ErrCode = "200010003"
	CodeClientNotInit ErrCode = "200010004"

	// Component: ProbeTool (20002xxxx)
	CodeInvalidLogInput ErrCode = "200020001"

	// Component: External/HttpClient (20250xxxx)
	CodeHttpSendFailed ErrCode = "202503002"

	// Component: Endpoint/Http (202501xxxx)
	CodeRequestDecodingFailed   ErrCode = "202501001"
	CodeRequiredFieldMissing    ErrCode = "202501002"
	CodeFieldConversionFailed   ErrCode = "202501003"
	CodeInvalidMappingFormat    ErrCode = "202501004"
	CodeDataTItemCreationFailed ErrCode = "202501005"

	// Component: External/RedisClient (202502xxxx)
	CodeRedisParseDSNFailed ErrCode = "202502001"
	CodeRedisConnectFailed  ErrCode = "202502002"

	// Component: External/DBClient (202503xxxx)
	CodeDBConnectFailed ErrCode = "202503001"

	// Component: External/HttpClient (202504xxxx)
	CodeHttpClientBuildRequestFailed ErrCode = "202504001"
	CodeHttpClientSendFailed         ErrCode = "202504002"
	CodeHttpClientInvalidProxy       ErrCode = "202504003"
	CodeHttpClientMapResponseFailed  ErrCode = "202504004"

	// Component: Asset (20003xxxx)
	CodeAssetUnmarshalFailed     ErrCode = "200030001"
	CodeAssetInvalidURI          ErrCode = "200030002"
	CodeAssetSchemeNotRegistered ErrCode = "200030003"
	CodeAssetTypeMismatch        ErrCode = "200030004"
	CodeAssetEmptyURI            ErrCode = "200030005"
	CodeAssetCannotSetForNonURI  ErrCode = "200030006"
	CodeAssetNotFound            ErrCode = "200030007"
)
