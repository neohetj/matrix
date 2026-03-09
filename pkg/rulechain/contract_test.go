package rulechain

import "testing"

func TestParseDataContractSkipsInternalPlaceholderObjIDs(t *testing.T) {
	contract := ParseDataContract(RawContract{
		Reads: []string{
			"rulemsg://dataT/<OptionalNotBound>?sid=String",
			"rulemsg://dataT/%3COptionalNotBound%3E?sid=%5B%5DAny",
			"rulemsg://dataT/real_input?sid=DiscoveryTask_V1",
		},
		Writes: []string{
			"rulemsg://dataT/<RequiredNotBound>?sid=String",
			"rulemsg://dataT/%3COutputNotBound%3E?sid=String",
			"rulemsg://dataT/real_output?sid=SliceString",
		},
	})

	if _, ok := contract.Reads["<OptionalNotBound>"]; ok {
		t.Fatalf("expected internal placeholder read objID to be ignored")
	}
	if _, ok := contract.Reads["%3COptionalNotBound%3E"]; ok {
		t.Fatalf("expected encoded internal placeholder read objID to be ignored")
	}
	if _, ok := contract.Writes["<RequiredNotBound>"]; ok {
		t.Fatalf("expected internal placeholder write objID to be ignored")
	}
	if _, ok := contract.Writes["%3COutputNotBound%3E"]; ok {
		t.Fatalf("expected encoded internal placeholder write objID to be ignored")
	}
	if _, ok := contract.Reads["real_input"]; !ok {
		t.Fatalf("expected real_input read to be preserved")
	}
	if _, ok := contract.Writes["real_output"]; !ok {
		t.Fatalf("expected real_output write to be preserved")
	}
	if got := contract.ReadObjectTypes["real_input"]; got != "DiscoveryTask_V1" {
		t.Fatalf("expected read sid DiscoveryTask_V1, got %q", got)
	}
	if got := contract.WriteObjectTypes["real_output"]; got != "SliceString" {
		t.Fatalf("expected write sid SliceString, got %q", got)
	}
}
