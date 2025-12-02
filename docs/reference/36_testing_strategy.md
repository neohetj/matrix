---
uuid: "f2a1b0c9-d8e7-4f6a-9b5c-4d3e2f1a0b9c"
type: "TestingStrategy"
title: "学习Matrix框架测试策略"
status: "Draft"
owner: "@Matrix-Core-Team"
version: "1.0.0"
tags:
  - "testing"
  - "strategy"
  - "unit-test"
  - "integration-test"
relations:
  - type: "is_referenced_by"
    target_uuid: "a2c8d4e1-7b3e-4c2a-8f5d-9e1b3c4d5a6b" # -> Matrix节点/组件开发SOP
    description: "The node development SOP requires developers to follow the testing strategy outlined here."
---

# 如何为Matrix框架贡献测试 (TestingStrategy)

本文档定义了为Matrix框架及其组件编写测试的官方策略和最佳实践。所有代码贡献都必须遵循本指南以确保代码质量和稳定性。

## 1. 核心测试理念 (CorePhilosophy)

Matrix框架的测试策略遵循分层测试金字塔模型，主要包括三个层面：
1.  **单元测试 (Unit Tests)**: 针对单个节点或函数的逻辑进行快速、隔离的验证。
2.  **集成测试 (Integration Tests)**: 针对一条完整的规则链，验证多个节点协同工作的正确性。
3.  **端到端测试 (End-to-End Tests)**: (未来规划) 针对承载Matrix核心引擎的完整应用（如Trinity）进行黑盒测试。

## 2. 单元测试最佳实践 (UnitTestingBestPractices)

### 2.1. 优先使用依赖注入进行Mock (PreferDependencyInjectionForMocking)

**核心原则**: 将被测单元与其外部依赖（网络、数据库、文件系统等）完全隔离。

*   **[强制]** 当被测函数依赖于外部服务（如HTTP客户端、数据库连接）时，**禁止**在单元测试中进行真实的网络或I/O调用。
*   **[推荐]** 最佳实践是将被测代码重构为依赖于一个**接口**，而不是一个具体的实现。例如，`external/httpClient` 节点依赖 `httpDoer` 接口，而不是 `*http.Client` 结构体。
*   在测试中，可以提供一个实现了该接口的**Mock对象**，从而完全控制被测函数的输入和外部调用的输出，使测试更加稳定和可预测。

**示例：**
```go
// 在被测代码中
type HttpClientNode struct {
    // ...
    client httpDoer // 依赖接口
}

// 在测试代码中
type mockHttpDoer struct {
    doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHttpDoer) Do(req *http.Request) (*http.Response, error) {
    return m.doFunc(req)
}

func TestMyNode(t *testing.T) {
    node := &HttpClientNode{...}
    node.client = &mockHttpDoer{ // 注入Mock对象
        doFunc: func(req *http.Request) (*http.Response, error) {
            // 返回预设的响应或错误
            return &http.Response{...}, nil
        },
    }
    // ... 执行测试 ...
}
```

### 2.2. 使用通用测试工具 (UseCommonTestUtilities)

为了简化测试的编写并保持一致性，Matrix在 `matrix/test` 包中提供了一系列通用的测试辅助工具。

*   **位置**: `matrix/test/`
*   **核心工具**:
    *   `test.MockNodeCtx`: 一个 `types.NodeCtx` 的Mock实现，用于捕获节点的输出 (`TellSuccess`, `TellFailure`) 以便在测试中断言。
    *   `test.TestLogger`: 一个简单的 `types.Logger` 实现，它会将日志打印到标准输出，方便在 `go test` 运行时查看日志。
    *   `test.MockLogger`: 一个 `types.Logger` 的Mock实现，它会捕获日志输出到一个内部缓冲区，用于对日志内容进行断言。
    *   `test.NewTestRuleMsg()`: 一个辅助函数，用于快速创建一个用于测试的、空的 `RuleMsg`。
    *   `test.GetRootError()`: 一个辅助函数，用于从一个嵌套的错误链中提取出根 `ErrorObj`。

**[推荐]** 在编写新的单元测试时，应优先使用这些在 `matrix/test` 中定义的工具。

*（本文档为骨架文件，详细内容待后续填充。）*

<!-- qa_section_start -->
> **问：我应该为我的新节点优先编写哪种测试？**
> **答：** 单元测试和集成测试都同样重要，缺一不可。你应该首先为节点内部的复杂业务逻辑编写单元测试，确保其在隔离环境下的正确性。然后，必须编写至少一个集成测试，将你的节点放入一个真实的规则链中，验证其在实际运行环境中的行为是否符合预期。
<!-- qa_section_end -->
