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

	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	HttpClientNodeType = "external/httpClient"
)

var (
	ErrHttpSendFailed = &types.ErrorObj{Code: 202503002, Message: "HTTP request sending failed"}
)

var httpClientNodePrototype = &HttpClientNode{
	BaseNode: *types.NewBaseNode(HttpClientNodeType, types.NodeDefinition{
		Name:        "HTTP Client",
		Description: "Sends a highly configurable HTTP request based on declarative mappings.",
		Dimension:   "External",
		Tags:        []string{"external", "http", "rest", "api"},
		Version:     "3.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(httpClientNodePrototype)
}

// HttpClientNodeConfiguration holds the configuration for the HttpClientNode.
type HttpClientNodeConfiguration struct {
	DefaultTimeout string                 `json:"defaultTimeout"`
	ProxyURL       string                 `json:"proxyUrl"`
	Request        helper.HttpRequestMap  `json:"request"`
	Response       helper.HttpResponseMap `json:"response"`
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

func (n *HttpClientNode) Init(cfg types.Config) error {
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

	var check func(sid string) error
	check = func(sid string) error {
		if sid == "" {
			return nil
		}
		if _, ok := registry.Get(sid); !ok {
			return fmt.Errorf("SID '%s' is not registered in CoreObjRegistry", sid)
		}
		return nil
	}

	// Check response body mappings
	if respBody := n.nodeConfig.Response.Body; respBody != nil {
		if respBody.From != nil {
			if err := check(respBody.From.DefineSID); err != nil {
				return err
			}
		}
		for _, param := range respBody.Params {
			if err := check(param.Mapping.DefineSID); err != nil {
				return fmt.Errorf("in response body param '%s': %w", param.Name, err)
			}
		}
	}

	// Check response header mappings
	if respHeaders := n.nodeConfig.Response.Headers; respHeaders != nil {
		if respHeaders.From != nil {
			if err := check(respHeaders.From.DefineSID); err != nil {
				return err
			}
		}
		for _, param := range respHeaders.Params {
			if err := check(param.Mapping.DefineSID); err != nil {
				return fmt.Errorf("in response header param '%s': %w", param.Name, err)
			}
		}
	}

	return nil
}

func (n *HttpClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// --- Start Debug Logging ---
	ctx.Info("HttpClientNode received message")
	// --- End Debug Logging ---

	// 1. Build Http Request from RuleMsg using the shared helper
	httpReq, err := helper.MapRuleMsgToHttpRequest(ctx, msg, n.nodeConfig.Request, n.nodeConfig.DefaultTimeout)
	if err != nil {
		ctx.TellFailure(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("failed to build http request: %w", err)))
		return
	}

	// --- Start Debug Logging ---
	ctx.Info("HttpClientNode built request",
		"URL", httpReq.URL.String(),
		"Method", httpReq.Method,
		"Headers", httpReq.Header)
	// --- End Debug Logging ---

	// 2. Create a new client for each request to ensure thread safety and dynamic configuration.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if n.nodeConfig.ProxyURL != "" {
		proxyUrl, err := url.Parse(n.nodeConfig.ProxyURL)
		if err != nil {
			ctx.TellFailure(msg, types.ErrInvalidParams.Wrap(fmt.Errorf("invalid proxy url: %w", err)))
			return
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}
	// If the node's client is the default http.Client, update its transport.
	// This allows the default client to be used while still supporting dynamic proxy settings.
	if hc, ok := n.client.(*http.Client); ok {
		hc.Transport = transport
	}

	// 3. Send Request and measure latency
	startTime := time.Now()
	resp, err := n.client.Do(httpReq)
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
	if mapErr := helper.MapHttpResponseToRuleMsg(ctx, resp, outMsg, n.nodeConfig.Response, startTime, endTime, err); mapErr != nil {
		ctx.TellFailure(msg, types.ErrInternal.Wrap(fmt.Errorf("failed to map response to message: %w", mapErr)))
		return
	}

	// 5. Now, after mapping the error (if any), we can fail the message.
	if err != nil {
		ctx.TellFailure(outMsg, ErrHttpSendFailed.Wrap(err))
		return
	}

	ctx.TellSuccess(outMsg)
}

func (n *HttpClientNode) Destroy() {}
