# 端到端测试计划：HTTP Client 字段覆盖与补充特性验证

## 目标
验证 `external/httpClient` 节点中 `Request.Body` 映射的 `From`（整体映射）与 `Params`（逐个映射）的组合行为。具体验证：
1.  **覆盖（Override）**：`Params` 中定义的字段能够覆盖 `From` 映射的动态源中的同名字段。
2.  **补充（Supplement）**：`Params` 中定义的字段能够作为新字段添加到 `From` 映射的基础结构中。

## 场景：告警增强转发
模拟一个告警转发场景，规则链将接收到的原始告警转发给外部系统。在此过程中：
*   **整体映射**：将原始告警对象（`parsedAlert`）作为基础请求体。
*   **字段覆盖**：将原始告警中的 `labels.severity` 字段值强制修改为 "CRITICAL_URGENT"（模拟业务规则的调整）。
*   **字段补充**：在请求体中新增一个 `forwarded_at` 字段，记录转发时间（模拟元数据注入）。

## 实施步骤

### 1. 创建新的规则链配置 (`Matrix/test/e2e_alert/rulechains/e2e_alert_override.json`)
*   基于 `e2e_alert_webhook.json` 修改。
*   ID: `rc-e2e-alert-override`。
*   `external/httpClient` 节点配置调整：
    ```json
    "body": {
      "from": {
        "path": "dataT.asdflasjgie", // 原始告警对象
        "defineSid": "parsedAlert"
      },
      "params": [
        {
          "name": "labels.severity", // 目标字段路径（支持点分）
          "type": "string",
          "mapping": {
            "from": "'CRITICAL_URGENT'" // 静态值源（覆盖）
          }
        },
        {
          "name": "forwarded_at", // 目标字段路径
          "type": "int64",
          "mapping": {
            "from": "metadata.ts" // 动态值源（补充）
          }
        }
      ]
    }
    ```

### 2. 改进代码支持深层覆盖 (`Matrix/pkg/helper/http_mapper.go`)
目前的 `buildRequestBody` 实现只是简单赋值 `bodyMap[param.Name] = val`，不支持点分路径的深层设置。
需要将其修改为使用 `utils.SetValueByDotPath(bodyMap, param.Name, val)`，以支持如 `labels.severity` 这样的嵌套字段覆盖。

### 3. 更新 E2E 测试代码 (`Matrix/test/e2e_alert_processing_test.go`)
*   添加 `TestE2EAlertOverride`。
*   **测试逻辑**：
    1.  启动模拟服务器。
    2.  构造初始 `RuleMsg`（包含 `metadata.ts`）。
    3.  启动规则链 `rc-e2e-alert-override`。
    4.  发送一个告警（`severity: "critical"`）。
    5.  断言模拟服务器收到的 Request Body：
        *   `labels.severity` 应为 "CRITICAL_URGENT"（验证覆盖）。
        *   存在 `forwarded_at` 字段且值正确（验证补充）。
        *   其他字段（如 `annotations.summary`）保持原样（验证整体映射保留）。
