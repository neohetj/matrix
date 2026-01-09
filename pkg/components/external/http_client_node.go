/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package external

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	HttpClientNodeType = "external/httpClient"
)

var (
	FaultHttpClientBuildRequestFailed = &types.Fault{Code: cnst.CodeHttpClientBuildRequestFailed, Message: "failed to build http request"}
	FaultHttpSendFailed               = &types.Fault{Code: cnst.CodeHttpClientSendFailed, Message: "HTTP request sending failed"}
	FaultHttpClientInvalidProxy       = &types.Fault{Code: cnst.CodeHttpClientInvalidProxy, Message: "invalid proxy url"}
	FaultHttpClientMapResponseFailed  = &types.Fault{Code: cnst.CodeHttpClientMapResponseFailed, Message: "failed to map response to message"}
)

var httpClientNodePrototype = &HttpClientNode{
	BaseNode: *types.NewBaseNode(HttpClientNodeType, types.NodeMetadata{
		Name:        "HTTP Client",
		Description: "Sends a highly configurable HTTP request based on declarative mappings.",
		Dimension:   "External",
		Tags:        []string{"external", "http", "rest", "api"},
		Version:     "1.0.0",
	}),
}

func init() {
	types.DefaultRegistry.GetNodeManager().Register(httpClientNodePrototype)
	types.DefaultRegistry.GetFaultRegistry().Register(
		FaultHttpClientBuildRequestFailed,
		FaultHttpSendFailed,
		FaultHttpClientInvalidProxy,
		FaultHttpClientMapResponseFailed,
	)
}

// HttpClientNodeConfiguration holds the configuration for the HttpClientNode.
type HttpClientNodeConfiguration struct {
	DefaultTimeout string                `json:"defaultTimeout"`
	ProxyURL       string                `json:"proxyUrl"`
	Request        types.HttpRequestMap  `json:"request"`
	Response       types.HttpResponseMap `json:"response"`
}

// httpDoer is an interface that wraps the Do method of an http.Client.
// This allows for mocking in tests.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HttpClientNode is a component that provides a configurable HTTP client.
type HttpClientNode struct {
	types.BaseNode
	types.Instance
	nodeConfig HttpClientNodeConfiguration
	// client is the underlying HTTP client. It is an interface to allow mocking.
	client httpDoer
}

func (n *HttpClientNode) New() types.Node {
	return &HttpClientNode{
		BaseNode: n.BaseNode,
		client:   &http.Client{}, // Default client
	}
}

func (n *HttpClientNode) Init(cfg types.ConfigMap) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode http client node config: %w", err)
	}
	if err := n.validateDefineSIDs(); err != nil {
		return fmt.Errorf("SID validation failed: %w", err)
	}
	return nil
}

// validateDefineSIDs checks if all defineSid fields in the configuration
// correspond to a registered CoreObj definition.
func (n *HttpClientNode) validateDefineSIDs() error {
	registry := registry.Default.GetCoreObjRegistry()

	check := func(sid string) error {
		if sid == "" {
			return nil
		}
		if _, ok := registry.Get(sid); !ok {
			return fmt.Errorf("SID '%s' is not registered in CoreObjRegistry", sid)
		}
		return nil
	}

	// Helper to check packet SIDs
	checkPacket := func(packet types.EndpointIOPacket, context string) error {
		if packet.MapAll != nil && *packet.MapAll != "" {
			if u, err := url.Parse(*packet.MapAll); err == nil && u.Scheme == "rulemsg" {
				if sid := u.Query().Get("sid"); sid != "" {
					if err := check(sid); err != nil {
						return fmt.Errorf("in %s MapAll: %w", context, err)
					}
				}
			}
		}
		for _, field := range packet.Fields {
			if field.BindPath != "" {
				if u, err := url.Parse(field.BindPath); err == nil && u.Scheme == "rulemsg" {
					if sid := u.Query().Get("sid"); sid != "" {
						if err := check(sid); err != nil {
							return fmt.Errorf("in %s field '%s': %w", context, field.Name, err)
						}
					}
				}
			}
		}
		return nil
	}

	// Check response body mappings
	if err := checkPacket(n.nodeConfig.Response.Body, "response body"); err != nil {
		return err
	}

	// Check response header mappings
	if err := checkPacket(n.nodeConfig.Response.Headers, "response headers"); err != nil {
		return err
	}

	return nil
}

func (n *HttpClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// --- Start Debug Logging ---
	ctx.Info("HttpClientNode received message")
	// --- End Debug Logging ---

	// 1. Build Http Request from RuleMsg using the shared helper
	httpReq, cancel, err := helper.MapRuleMsgToHttpRequest(ctx, msg, n.nodeConfig.Request, n.nodeConfig.DefaultTimeout)
	if err != nil {
		ctx.HandleError(msg, FaultHttpClientBuildRequestFailed.Wrap(err))
		return
	}
	defer cancel()

	// --- Start Debug Logging ---
	ctx.Info("HttpClientNode built request",
		"URL", httpReq.URL.String(),
		"Method", httpReq.Method,
		"Headers", httpReq.Header)
	// --- End Debug Logging ---

	// 2. Create a new client for each request to ensure thread safety and dynamic configuration.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Clone transport to avoid race conditions and side effects on global transport
	if n.nodeConfig.ProxyURL != "" {
		proxyUrl, err := url.Parse(n.nodeConfig.ProxyURL)
		if err != nil {
			ctx.HandleError(msg, FaultHttpClientInvalidProxy.Wrap(err))
			return
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	// Create a new client instance for this request to use the configured transport.
	// We check if n.client is a *http.Client to copy its other settings (like Timeout) if needed,
	// but here we primarily need to ensure we use the potentially modified transport.
	// Note: We are creating a NEW client instance here, not modifying the shared n.client.
	// However, if n.client is a mock (does not implement *http.Client), we should use it directly
	// (though mocks usually don't need proxy settings).
	var requestClient httpDoer
	if hc, ok := n.client.(*http.Client); ok {
		// Create a shallow copy of the client to swap the transport
		newClient := *hc
		newClient.Transport = transport
		requestClient = &newClient
	} else {
		// It's a mock or custom implementation, use as is
		requestClient = n.client
	}

	// 3. Send Request and measure latency
	startTime := time.Now()
	resp, err := requestClient.Do(httpReq)
	endTime := time.Now()

	// We handle the error after mapping, so we can record it.
	if err == nil {
		defer resp.Body.Close()
	}

	// 3. Create a new message for the output.
	// This is crucial because this node modifies the message's Data field with the HTTP response body.
	// msg.Copy() performs a shallow copy: Metadata is duplicated, but the DataT object is a shared reference.
	// This protects the original msg.Data from being overwritten, which is important if other parallel
	// branches in the rule chain need the original data. For true DataT isolation, a msg.DataT().DeepCopy()
	// would be required.
	outMsg := msg.Copy()

	// 4. Map Response back to the new Message using the shared helper
	// Pass requestErr (err) to MapHttpResponseToRuleMsg. If requestErr is not nil,
	// MapHttpResponseToRuleMsg will map it to metadata and return nil (no mapping error).
	// We check for requestErr later to decide whether to fail the node execution.
	if mapErr := helper.MapHttpResponseToRuleMsg(ctx, resp, outMsg, n.nodeConfig.Response, startTime, endTime, err); mapErr != nil {
		ctx.HandleError(msg, FaultHttpClientMapResponseFailed.Wrap(mapErr))
		return
	}

	// 5. Now, after mapping the error (if any), we can fail the message.
	if err != nil {
		ctx.HandleError(outMsg, FaultHttpSendFailed.Wrap(err))
		return
	}

	ctx.TellSuccess(outMsg)
}

func (n *HttpClientNode) Destroy() {}

// Errors returns the list of possible faults that this node can produce.
func (n *HttpClientNode) Errors() []*types.Fault {
	return []*types.Fault{
		FaultHttpClientBuildRequestFailed,
		FaultHttpSendFailed,
		FaultHttpClientInvalidProxy,
		FaultHttpClientMapResponseFailed,
	}
}
