# MobiFlight WASM 模块 Go 使用指南

## 目录
1. [简介](#简介)
2. [快速开始](#快速开始)
3. [核心概念](#核心概念)
4. [API 参考](#api-参考)
5. [示例代码](#示例代码)
6. [常见问题](#常见问题)
7. [故障排除](#故障排除)

---

## 简介

本项目提供了一个 Go 语言封装，用于通过 MobiFlight WASM 模块与 Microsoft Flight Simulator 进行通信。它允许您读取和设置飞机的本地变量（LVars），以及执行自定义的 RPN 计算器代码。

### 主要功能

- ✅ 读取飞机 LVar（本地变量）
- ✅ 设置飞机 LVar 的值
- ✅ 批量操作多个 LVar
- ✅ 执行自定义 RPN 计算器代码
- ✅ 异步消息处理
- ✅ 完整的错误处理

---

## 快速开始

### 前提条件

1. **Microsoft Flight Simulator** 已安装并运行
2. **MobiFlight WASM 模块** 已安装在模拟器中
   - 下载地址: https://www.mobiflight.com/
3. **SimConnect.dll** 位于 `simconnect/` 目录
4. **Go 1.24.7** 或更高版本

### 安装

```bash
# 克隆仓库
git clone https://github.com/NovaVoyager/Microsoft_Simconnect.git
cd Microsoft_Simconnect

# 确保 SimConnect.dll 存在
ls simconnect/SimConnect.dll
```

### 第一个程序

创建 `my_first_app.go`:

```go
package main

import (
    "fmt"
    "log"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    // 1. 初始化客户端
    sc, mf, err := simconnect.SetupMobiFlightClient("我的第一个应用")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    fmt.Println("✓ 连接成功!")

    // 2. 设置一个 LVar
    err = mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 1.0)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("✓ LVar 设置成功!")
}
```

运行:
```bash
go run my_first_app.go
```

---

## 核心概念

### 1. LVar (Local Variables)

LVar 是飞机模型特定的本地变量。每个飞机可能有不同的 LVar。

**命名约定:**
- 通常以 `L:` 开头（在代码中可以省略）
- 例如: `L:XMLVAR_Switch_AP_MSL_Mode`

**常见 LVar 示例:**

| LVar 名称 | 说明 | 典型值 |
|-----------|------|--------|
| `XMLVAR_Switch_AP_MSL_Mode` | 自动驾驶高度模式 | 0=关闭, 1=开启 |
| `XMLVAR_Autopilot_Altitude` | 自动驾驶目标高度 | 0-50000 (英尺) |
| `XMLVAR_Airspeed_Mode` | 空速模式 | 0-3 |
| `XMLVAR_Switch_EFIS_Range` | EFIS 显示范围 | 0-5 |

### 2. RPN 计算器代码

RPN (Reverse Polish Notation) 是一种后缀表达式，用于在模拟器中执行计算。

**基本语法:**

```
值 操作符
```

**示例:**

| RPN 代码 | 说明 |
|----------|------|
| `1 (>L:MY_VAR)` | 将 1 写入 L:MY_VAR |
| `(L:MY_VAR)` | 读取 L:MY_VAR |
| `100 200 +` | 计算 100 + 200 = 300 |
| `(L:VAR1) (L:VAR2) +` | 读取两个变量并相加 |
| `5 (>L:VAR1) 10 (>L:VAR2)` | 设置多个变量 |

**操作符:**
- `+` 加法
- `-` 减法
- `*` 乘法
- `/` 除法
- `>` 写入
- `<` 读取

### 3. Client Data Area (CDA)

SimConnect 使用客户端数据区域在应用程序和 WASM 模块之间传递数据。

**MobiFlight 使用的 CDA:**
- `MobiFlight.Command` - 发送命令
- `MobiFlight.Response` - 接收响应

---

## API 参考

### SimConnect 基础 API

#### `LoadDLL() (*SimConnect, error)`

加载 SimConnect.dll。

```go
sc, err := simconnect.LoadDLL()
if err != nil {
    log.Fatal(err)
}
```

#### `Open(clientName string) error`

打开与模拟器的连接。

```go
err := sc.Open("我的应用")
```

#### `Close() error`

关闭连接。

```go
defer sc.Close()
```

### MobiFlight 客户端 API

#### `SetupMobiFlightClient(clientName string) (*SimConnect, *MobiFlightClient, error)`

一站式初始化函数。

```go
sc, mf, err := simconnect.SetupMobiFlightClient("我的应用")
if err != nil {
    log.Fatal(err)
}
defer sc.Close()
```

#### `GetLVar(varName string) error`

请求读取 LVar 的值。

```go
err := mf.GetLVar("XMLVAR_Autopilot_Altitude")
// 需要在消息循环中调用 ReceiveResponse() 来获取结果
```

#### `SetLVar(varName string, value float64) error`

设置 LVar 的值。

```go
err := mf.SetLVar("XMLVAR_Autopilot_Altitude", 10000.0)
```

#### `ExecuteCode(code string) error`

执行 RPN 计算器代码。

```go
err := mf.ExecuteCode("100 200 + (>L:MY_VAR)")
```

#### `ReceiveResponse() (*MobiFlightResponse, error)`

从消息队列中接收响应。

**返回值说明：**
- `response != nil, err == nil` - 成功收到响应
- `response == nil, err == nil` - 当前没有消息（继续轮询）
- `response == nil, err != nil` - 发生错误

```go
response, err := mf.ReceiveResponse()
if err != nil {
    // 真正的错误
    log.Fatal(err)
}
if response == nil {
    // 没有消息，继续等待
    return
}
// 处理响应
value, _ := simconnect.ParseResponseFloat(response)
```

### 批量操作 API

#### `NewLVarBatch(client *MobiFlightClient) *LVarBatch`

创建批量操作对象。

```go
batch := simconnect.NewLVarBatch(mf)
```

#### `Add(name string, value float64)`

添加 LVar 到批量操作。

```go
batch.Add("XMLVAR_Switch_AP_MSL_Mode", 1.0)
batch.Add("XMLVAR_Autopilot_Altitude", 5000.0)
```

#### `Execute() error`

执行批量操作。

```go
err := batch.Execute()
```

### 辅助函数

#### `ParseResponseFloat(response *MobiFlightResponse) (float64, error)`

从响应中解析浮点数。

```go
value, err := simconnect.ParseResponseFloat(response)
```

#### `ParseResponseString(response *MobiFlightResponse) string`

从响应中解析字符串。

```go
str := simconnect.ParseResponseString(response)
```

---

## 示例代码

### 示例 1: 简单的读写操作

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    // 初始化
    sc, mf, err := simconnect.SetupMobiFlightClient("读写示例")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    // 设置 LVar
    fmt.Println("设置自动驾驶高度为 10000 英尺...")
    err = mf.SetLVar("XMLVAR_Autopilot_Altitude", 10000.0)
    if err != nil {
        log.Fatal(err)
    }

    time.Sleep(500 * time.Millisecond)

    // 读取 LVar
    fmt.Println("读取自动驾驶高度...")
    err = mf.GetLVar("XMLVAR_Autopilot_Altitude")
    if err != nil {
        log.Fatal(err)
    }

    // 等待并接收响应
    time.Sleep(200 * time.Millisecond)
    response, err := mf.ReceiveResponse()
    if err != nil {
        log.Printf("接收错误: %v", err)
    } else if response != nil {
        value, _ := simconnect.ParseResponseFloat(response)
        fmt.Printf("当前高度: %.0f 英尺\n", value)
    } else {
        fmt.Println("暂时没有收到响应")
    }
}
```

### 示例 2: 批量设置多个参数

```go
package main

import (
    "log"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    sc, mf, err := simconnect.SetupMobiFlightClient("批量操作示例")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    // 创建批量操作
    batch := simconnect.NewLVarBatch(mf)

    // 添加多个 LVar
    batch.Add("XMLVAR_Switch_AP_MSL_Mode", 1.0)
    batch.Add("XMLVAR_Autopilot_Altitude", 5000.0)
    batch.Add("XMLVAR_Airspeed_Mode", 2.0)

    // 一次性执行
    err = batch.Execute()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("批量设置完成!")
}
```

### 示例 3: 使用 RPN 代码

```go
package main

import (
    "log"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    sc, mf, err := simconnect.SetupMobiFlightClient("RPN 示例")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    // 执行 RPN 代码: 计算 5000 + 2500 并存储到 LVar
    rpnCode := "5000 2500 + (>L:MY_ALTITUDE)"
    err = mf.ExecuteCode(rpnCode)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("RPN 代码执行完成! L:MY_ALTITUDE 现在应该是 7500")
}
```

### 示例 4: 完整的消息循环

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    sc, mf, err := simconnect.SetupMobiFlightClient("消息循环示例")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    // 启动消息处理 goroutine
    responses := make(chan *simconnect.MobiFlightResponse, 10)
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()

        for range ticker.C {
            response, err := mf.ReceiveResponse()
            if err != nil {
                // 真正的错误
                log.Printf("接收错误: %v", err)
                continue
            }
            if response != nil {
                // 收到响应
                responses <- response
            }
            // response == nil 表示没有消息，继续轮询
        }
    }()

    // 请求 LVar
    err = mf.GetLVar("XMLVAR_Autopilot_Altitude")
    if err != nil {
        log.Fatal(err)
    }

    // 等待响应
    select {
    case response := <-responses:
        value, _ := simconnect.ParseResponseFloat(response)
        fmt.Printf("收到响应: %.0f\n", value)
    case <-time.After(5 * time.Second):
        fmt.Println("超时")
    }
}
```

---

## 常见问题

### Q1: 如何找到我的飞机支持的 LVar？

**答:** 使用以下工具之一:

1. **MobiFlight Connector** - 内置 LVar 查看器
2. **FSUIPC** - 提供 LVar 监控功能
3. **飞机文档** - 某些飞机提供 LVar 列表

### Q2: 为什么 GetLVar() 没有返回值？

**答:** `GetLVar()` 是异步操作。您需要:
1. 调用 `GetLVar()` 发送请求
2. 在消息循环中调用 `ReceiveResponse()` 接收响应
3. 使用 `ParseResponseFloat()` 解析响应

### Q3: 批量操作有什么优势？

**答:** 批量操作的优势:
- **性能**: 减少 SimConnect 调用次数
- **原子性**: 多个操作在一个事务中完成
- **效率**: 降低通信开销

### Q4: RPN 代码的执行顺序是什么？

**答:** RPN 使用栈式执行:
```
5 3 +     →  栈: [5] → [5, 3] → [8]
(>L:VAR)  →  从栈中弹出 8，写入 L:VAR
```

### Q5: 如何处理连接断开？

**答:** 实现重连逻辑:

```go
func connectWithRetry(maxRetries int) (*simconnect.SimConnect, error) {
    for i := 0; i < maxRetries; i++ {
        sc, _, err := simconnect.SetupMobiFlightClient("我的应用")
        if err == nil {
            return sc, nil
        }
        time.Sleep(time.Second * 2)
    }
    return nil, fmt.Errorf("连接失败")
}
```

---

## 故障排除

### 问题: SimConnect_Open failed, HRESULT=0x...

**可能原因:**
1. Microsoft Flight Simulator 未运行
2. SimConnect.dll 版本不匹配
3. 防火墙阻止连接

**解决方案:**
1. 确保模拟器正在运行
2. 从模拟器安装目录复制正确版本的 SimConnect.dll
3. 检查防火墙设置

### 问题: 没有收到 LVar 响应

**可能原因:**
1. LVar 名称错误
2. MobiFlight WASM 模块未加载
3. 消息循环未正确实现

**解决方案:**
1. 使用 MobiFlight Connector 验证 LVar 名称
2. 检查 WASM 模块是否在模拟器中加载
3. 实现正确的消息循环

### 问题: SetLVar() 不生效

**可能原因:**
1. LVar 是只读的
2. 值超出范围
3. 飞机不支持该 LVar

**解决方案:**
1. 查阅飞机文档确认 LVar 可写
2. 检查值的有效范围
3. 使用 MobiFlight Connector 测试

### 调试技巧

1. **启用 SimConnect 日志:**
   - 在模拟器目录创建 `SimConnect.xml`
   - 设置日志级别为 Verbose

2. **使用 MobiFlight Connector:**
   - 先用 MobiFlight Connector 测试 LVar
   - 确认参数和值都正确

3. **添加详细日志:**
   ```go
   log.Printf("发送命令: %s", command)
   log.Printf("接收响应: %+v", response)
   ```

4. **检查 HRESULT 错误码:**
   ```
   0x00000000 = 成功
   0x887A0001 = 模拟器未运行
   0x887A0005 = 连接超时
   ```

---

## 更多资源

- [MobiFlight 官方网站](https://www.mobiflight.com/)
- [SimConnect SDK 文档](https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/SimConnect_SDK.htm)
- [RPN 计算器参考](https://docs.flightsimulator.com/html/Additional_Information/Reverse_Polish_Notation.htm)
- [项目 GitHub](https://github.com/NovaVoyager/Microsoft_Simconnect)

---

## 许可证

请参阅 LICENSE 文件了解详情。
