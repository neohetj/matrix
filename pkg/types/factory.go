package types

import "context"

// Factory functions that can be replaced by other implementations.
var (
	// NewNodeCtx creates a new node context.
	NewNodeCtx func(ctx context.Context, r Runtime, chain ChainInstance, selfDef *NodeDef, parent NodeCtx, onEnd func(msg RuleMsg, err error), aspects []Aspect, callback CallbackFunc) NodeCtx
	// NewMsg creates a new message with the given type, data, and metadata.
	NewMsg func(msgType, data string, metadata Metadata, dataT DataT) RuleMsg
	// CloneMsgWithDataT creates a new message that preserves the source message identity
	// while replacing the structured DataT payload.
	CloneMsgWithDataT func(msg RuleMsg, dataT DataT) RuleMsg
	// NewSubMsg creates a new sub-message from a parent message.
	NewSubMsg func(parentMsg RuleMsg, subChainId string) RuleMsg
	// NewDataT creates a new instance of the default DataT implementation.
	NewDataT func() DataT

	// NewCoreObj creates a new CoreObj.
	NewCoreObj func(key string, def CoreObjDef) CoreObj
	// NewCoreObjDef creates a new CoreObjDef.
	NewCoreObjDef func(prototype any, sid, desc string) CoreObjDef
)
