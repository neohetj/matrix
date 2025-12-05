# 端到端测试计划：集成 HTTP Client 节点

## 目标
创建一个新的规则链，集成 HTTP Client 节点，并在现有的 E2E 告警处理测试套件中验证其功能。

## 场景：外部告警通知 (Webhook)
我们将模拟一个场景，其中严重（Critical）级别的告警需要通过 HTTP POST 转发到外部系统（例如 PagerDuty 或 Webhook 接收器）。

### 工作流
1.  **触发**：HTTP Endpoint 接收到一个告警（严重性：Critical）。
2.  **处理**：规则链解析并校验告警数据。
3.  **路由**：规则链识别该告警为 "Critical"。
4.  **动作**：规则链将严重告警路由到一个 `external/httpClient` 节点。
5.  **外部调用**：HTTP Client 节点向模拟的外部服务器发送 POST 请求。
6.  **验证**：测试代码验证模拟服务器是否收到了包含正确数据的请求。

## 实施步骤

### 1. 创建新的规则链配置 (`Matrix/test/e2e_alert/rulechains/e2e_alert_webhook.json`)
*   基于 `e2e_alert_processing.json` 创建。
*   ID 修改为 `rc-e2e-alert-webhook`。
*   添加一个名为 `node-webhook-notification` 的 `external/httpClient` 节点。
*   配置该节点：
    *   **URL**: `{{metadata.webhook_url}}/webhook` (将在测试中动态注入)。
    *   **Method**: `POST`.
    *   **Request Mapping**: 将 `dataT.asdflasjgie` (即 parsedAlert) 的数据映射到请求体。
*   更新连接：将 `node-route-by-severity` 的 `Critical` 路由指向 `node-webhook-notification`。

### 2. 更新 E2E 测试代码 (`Matrix/test/e2e_alert_processing_test.go`)
*   添加一个新的测试函数 `TestE2EAlertWebhook`。
*   **模拟服务器**：使用 `httptest` 创建一个本地 HTTP 服务器，模拟外部 Webhook 接收端。
*   **配置注入**：
    *   `HttpClientNode` 支持占位符。我们将模拟服务器的 URL 放入消息元数据（例如 `metadata.webhook_url`）中。
    *   节点配置中的 URL 设置为 `{{metadata.webhook_url}}/webhook`。
*   **测试逻辑**：
    1.  启动 `httptest` 服务器。
    2.  构造初始 `RuleMsg`，并在 Metadata 中设置 `webhook_url` 为模拟服务器的 Base URL。
    3.  启动规则链 `rc-e2e-alert-webhook`。
    4.  发送一个严重告警。
    5.  断言模拟服务器接收到了 POST 请求，且请求体与预期一致。

## 新规则链 JSON 结构预览 (`e2e_alert_webhook.json`)
```json
{
  "ruleChain": {
    "id": "rc-e2e-alert-webhook",
    "name": "E2E Alert Webhook Rule Chain",
    ...
  },
  "metadata": {
    "nodes": [
      ...,
      {
        "id": "node-webhook-notification",
        "type": "external/httpClient",
        "name": "发送 Webhook",
        "configuration": {
          "request": {
            "url": "{{metadata.webhook_url}}/webhook",
            "method": "POST",
            "body": {
              "from": {
                "path": "dataT.asdflasjgie",
                "defineSid": "parsedAlert"
              }
            }
          },
          "response": {
              "statusCodeTarget": "metadata.webhook_status"
          }
        }
      }
    ],
    "connections": [
      ...,
      { "fromId": "node-route-by-severity", "toId": "node-webhook-notification", "type": "Critical" }
    ]
  }
}