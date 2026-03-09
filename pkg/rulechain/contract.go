package rulechain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// RawContract is a lightweight read/write contract shape that can be parsed from
// node contracts, DSL fragments, or graph payloads.
type RawContract struct {
	Reads  []string `json:"reads"`
	Writes []string `json:"writes"`
}

// ContractAccess is the normalized object-level contract model used by runtime
// projection and graph data-flow analysis.
type ContractAccess struct {
	Reads            map[string]struct{}
	Writes           map[string]struct{}
	ReadObjectTypes  map[string]string
	WriteObjectTypes map[string]string
	ReadDataFormats  map[string]string
	WriteDataFormats map[string]string
	DataRead         bool
	DataWrite        bool
	DataFullWrite    bool
	IsPassThrough    bool
	ConsumesAll      bool
}

// ParseDataContract normalizes multiple contract shapes into a single access model.
func ParseDataContract(contract any) ContractAccess {
	result := ContractAccess{
		Reads:            map[string]struct{}{},
		Writes:           map[string]struct{}{},
		ReadObjectTypes:  map[string]string{},
		WriteObjectTypes: map[string]string{},
		ReadDataFormats:  map[string]string{},
		WriteDataFormats: map[string]string{},
	}

	readList, writeList := extractReadWriteLists(contract)
	handle := func(uri string, isWrite bool) {
		if uri == "" {
			return
		}
		if uri == "rulemsg://*" {
			if !isWrite {
				result.ConsumesAll = true
			}
			return
		}

		if strings.HasPrefix(uri, "rulemsg://dataT/") {
			trimmed := strings.TrimPrefix(uri, "rulemsg://dataT/")
			parts := strings.SplitN(trimmed, "?", 2)
			pathPart := decodeQueryValue(parts[0])
			if pathPart == "" {
				return
			}
			objID := strings.Split(pathPart, ".")[0]
			if objID == "" || isInternalPlaceholderObjID(objID) {
				return
			}
			sid := ""
			if len(parts) == 2 {
				sid = decodeQueryValue(queryParam(parts[1], "sid"))
			}
			if isWrite {
				result.Writes[objID] = struct{}{}
				if sid != "" {
					result.WriteObjectTypes[objID] = sid
				}
			} else {
				result.Reads[objID] = struct{}{}
				if sid != "" {
					result.ReadObjectTypes[objID] = sid
				}
			}
			return
		}

		if strings.HasPrefix(uri, "rulemsg://data") {
			format := "JSON"
			parts := strings.Split(uri, "?")
			uriPath := strings.TrimPrefix(parts[0], "rulemsg://")
			dataPath := ""
			if strings.HasPrefix(uriPath, "data/") {
				dataPath = strings.TrimPrefix(uriPath, "data/")
			}
			if len(parts) > 1 {
				if f := queryParam(parts[1], "format"); f != "" {
					format = f
				}
			}
			if isWrite {
				result.DataWrite = true
				result.DataFullWrite = dataPath == ""
				result.WriteDataFormats[dataPath] = format
			} else {
				result.DataRead = true
				result.ReadDataFormats[dataPath] = format
			}
			return
		}

		if strings.HasPrefix(uri, "dataT.") {
			parts := strings.Split(uri, ".")
			if len(parts) > 1 && parts[1] != "" && !isInternalPlaceholderObjID(parts[1]) {
				if isWrite {
					result.Writes[parts[1]] = struct{}{}
				} else {
					result.Reads[parts[1]] = struct{}{}
				}
			}
			return
		}

		if uri == "data" || strings.HasPrefix(uri, "data.") {
			dataPath := ""
			if uri != "data" {
				dataPath = strings.TrimPrefix(uri, "data.")
			}
			if isWrite {
				result.DataWrite = true
				result.WriteDataFormats[dataPath] = "UNKNOWN"
			} else {
				result.DataRead = true
				result.ReadDataFormats[dataPath] = "UNKNOWN"
			}
		}
	}

	for _, item := range readList {
		handle(item, false)
	}
	for _, item := range writeList {
		handle(item, true)
	}

	if contains(readList, "rulemsg://*") && contains(writeList, "rulemsg://*") {
		result.IsPassThrough = true
	}

	return result
}

// BuildEndpointContract extracts endpoint read/write contracts from endpoint configuration.
func BuildEndpointContract(endpointConfiguration map[string]any) RawContract {
	contract := RawContract{Reads: []string{}, Writes: []string{}}
	if endpointConfiguration == nil {
		return contract
	}

	endpointDef, _ := endpointConfiguration["endpointDefinition"].(map[string]any)
	if endpointDef == nil {
		return contract
	}

	request, _ := endpointDef["request"].(map[string]any)
	response, _ := endpointDef["response"].(map[string]any)
	if request != nil {
		handlePacketToContract(request["body"], &contract.Writes)
		handlePacketToContract(request["queryParams"], &contract.Writes)
		handlePacketToContract(request["headers"], &contract.Writes)
		handlePathParamsToContract(request["pathParams"], &contract.Writes)
	}
	if response != nil {
		handlePacketToContract(response["body"], &contract.Reads)
		handlePacketToContract(response["headers"], &contract.Reads)
	}
	return contract
}

// BuildNodeContractAccess merges runtime DataContract() with DSL inputs/outputs for a node.
func BuildNodeContractAccess(node types.Node, def *types.NodeDef) ContractAccess {
	if node == nil {
		return ParseDataContract(nil)
	}
	contract := node.DataContract()
	result := ParseDataContract(RawContract{
		Reads:  contract.Reads,
		Writes: contract.Writes,
	})
	overlayNodeIODefinitions(&result, def)
	return result
}

// CollectChainContracts builds normalized contracts for all runtime nodes in a chain instance.
func CollectChainContracts(instance types.ChainInstance) map[string]ContractAccess {
	result := map[string]ContractAccess{}
	if instance == nil {
		return result
	}
	for id, node := range instance.GetAllNodes() {
		def, _ := instance.GetNodeDef(id)
		result[id] = BuildNodeContractAccess(node, def)
	}
	return result
}

// ContractAccessFromCoreObjSet turns a derived object set into a synthetic pass-through contract.
func ContractAccessFromCoreObjSet(set types.CoreObjSet) ContractAccess {
	if set.RetainAll {
		return ParseDataContract(RawContract{
			Reads:  []string{"rulemsg://*"},
			Writes: []string{"rulemsg://*"},
		})
	}
	raw := RawContract{Reads: make([]string, 0, len(set.ObjIDs)), Writes: make([]string, 0, len(set.ObjIDs))}
	for _, objID := range set.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID == "" {
			continue
		}
		uri := fmt.Sprintf("rulemsg://dataT/%s", objID)
		raw.Reads = append(raw.Reads, uri)
		raw.Writes = append(raw.Writes, uri)
	}
	return ParseDataContract(raw)
}

func overlayNodeIODefinitions(result *ContractAccess, def *types.NodeDef) {
	if result == nil || def == nil {
		return
	}
	overlayObjectTypeMap(result.Reads, result.ReadObjectTypes, def.Inputs)
	overlayObjectTypeMap(result.Writes, result.WriteObjectTypes, def.Outputs)
}

func overlayObjectTypeMap(target map[string]struct{}, typeMap map[string]string, raw map[string]any) {
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok || obj == nil {
			continue
		}
		objID, _ := obj["objId"].(string)
		sid, _ := obj["defineSid"].(string)
		objID = strings.TrimSpace(objID)
		sid = strings.TrimSpace(sid)
		if objID == "" || isInternalPlaceholderObjID(objID) {
			continue
		}
		target[objID] = struct{}{}
		if sid != "" && typeMap[objID] == "" {
			typeMap[objID] = sid
		}
	}
}

func extractReadWriteLists(contract any) (reads []string, writes []string) {
	reads = []string{}
	writes = []string{}

	appendStringArray := func(src any, target *[]string) {
		switch v := src.(type) {
		case []string:
			*target = append(*target, v...)
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					*target = append(*target, s)
				}
			}
		}
	}

	switch c := contract.(type) {
	case RawContract:
		reads = append(reads, c.Reads...)
		writes = append(writes, c.Writes...)
	case *RawContract:
		if c != nil {
			reads = append(reads, c.Reads...)
			writes = append(writes, c.Writes...)
		}
	case map[string]any:
		appendStringArray(c["Reads"], &reads)
		appendStringArray(c["reads"], &reads)
		appendStringArray(c["inputs"], &reads)
		appendStringArray(c["Writes"], &writes)
		appendStringArray(c["writes"], &writes)
		appendStringArray(c["outputs"], &writes)
	}

	return reads, writes
}

func isInternalPlaceholderObjID(objID string) bool {
	trimmed := decodeQueryValue(strings.TrimSpace(objID))
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">")
}

func queryParam(query string, key string) string {
	for _, pair := range strings.Split(query, "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

func decodeQueryValue(value string) string {
	if value == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func handlePacketToContract(packet any, target *[]string) {
	obj, ok := packet.(map[string]any)
	if !ok || obj == nil {
		return
	}
	if mapAll, ok := obj["mapAll"].(string); ok && mapAll != "" {
		appendBindPath(mapAll, target)
	}
	if fields, ok := obj["fields"].([]any); ok {
		for _, item := range fields {
			field, _ := item.(map[string]any)
			if field == nil {
				continue
			}
			if bindPath, ok := field["bindPath"].(string); ok && bindPath != "" {
				appendBindPath(bindPath, target)
			}
		}
	}
}

func handlePathParamsToContract(pathParams any, target *[]string) {
	if pathParams == nil {
		return
	}
	if arr, ok := pathParams.([]any); ok {
		for _, item := range arr {
			field, _ := item.(map[string]any)
			if field == nil {
				continue
			}
			if bindPath, ok := field["bindPath"].(string); ok && bindPath != "" {
				appendBindPath(bindPath, target)
			}
		}
		return
	}
	handlePacketToContract(pathParams, target)
}

func appendBindPath(bindPath string, target *[]string) {
	if bindPath == "" {
		return
	}
	if strings.HasPrefix(bindPath, "rulemsg://") {
		*target = append(*target, bindPath)
		return
	}
	if strings.HasPrefix(bindPath, "dataT.") {
		parts := strings.Split(bindPath, ".")
		if len(parts) > 1 && parts[1] != "" {
			*target = append(*target, fmt.Sprintf("rulemsg://dataT/%s", parts[1]))
		}
		return
	}
	if strings.HasPrefix(bindPath, "data.") {
		*target = append(*target, fmt.Sprintf("rulemsg://data/%s", strings.TrimPrefix(bindPath, "data.")))
		return
	}
	if bindPath == "data" {
		*target = append(*target, "rulemsg://data")
	}
}
