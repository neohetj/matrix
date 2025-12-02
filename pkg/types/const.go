package types

const (
	// Basic Type SIDs
	SID_STRING               = "string"
	SID_INT64                = "int64"
	SID_FLOAT64              = "float64"
	SID_BOOL                 = "bool"
	SID_MAP_STRING_STRING    = "map_string_string"
	SID_MAP_STRING_INTERFACE = "map_string_interface"
)

const (
	STRING  DataType = "string"
	INT     DataType = "int"
	INTEGER DataType = "integer"
	FLOAT   DataType = "float"
	DOUBLE  DataType = "double"
	NUMBER  DataType = "number"
	BOOL    DataType = "bool"
	BOOLEAN DataType = "boolean"
	OBJECT  DataType = "object"
	MAP     DataType = "map"
)

const (
	JSON    DataFormat = "JSON"
	TEXT    DataFormat = "TEXT"
	BYTES   DataFormat = "BYTES"
	UNKNOWN DataFormat = "UNKNOWN"
)

// ErrCode defines the type for standardized error codes.
type ErrCode int32

// Predefined error codes for the Matrix engine.
// The code format is aabbbcccc, where:
// aa: 20 (Software Product Department)
// bbb: Module (000: Global, 001: Runtime, 002: Parser, etc.)
// cccc: Specific error identifier.
const (
	// Global Errors (20000xxxx)
	CodeInternalError        ErrCode = 200000001
	CodeInvalidParams        ErrCode = 200000002
	CodeInvalidConfiguration ErrCode = 200000003

	// Runtime Errors (20001xxxx)
	CodeNodeNotFound ErrCode = 200010001
	CodeFuncNotFound ErrCode = 200010002

	// Component: ProbeTool (20002xxxx)
	CodeInvalidLogInput ErrCode = 200020001

	// Component: External/HttpClient (20250xxxx)
	CodeHttpSendFailed ErrCode = 202503002

	// Component: Endpoint/Http (202501xxxx)
	CodeRequestDecodingFailed   ErrCode = 202501001
	CodeRequiredFieldMissing    ErrCode = 202501002
	CodeFieldConversionFailed   ErrCode = 202501003
	CodeInvalidMappingFormat    ErrCode = 202501004
	CodeDataTItemCreationFailed ErrCode = 202501005
)
