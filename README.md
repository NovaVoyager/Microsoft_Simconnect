# Microsoft SimConnect Go 封装

一个用于 Microsoft Flight Simulator 的 Go 语言 SimConnect SDK 封装，支持通过 MobiFlight WASM 模块与飞机参数进行交互。

![Go Version](https://img.shields.io/badge/go-%3E%3D1.24.7-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## ✨ 特性

- ✅ **SimConnect 基础封装** - 完整的 SimConnect API 支持
- ✅ **MobiFlight WASM 支持** - 读写飞机 LVar（本地变量）
- ✅ **批量操作** - 高效的批量参数设置
- ✅ **RPN 计算器** - 执行自定义 RPN 代码
- ✅ **类型安全** - 完整的类型定义和错误处理
- ✅ **中文文档** - 详细的中文使用指南

## 📦 安装

### 前提条件

1. **Microsoft Flight Simulator** (MSFS 2020 或 2024)
2. **MobiFlight WASM 模块** - [下载地址](https://www.mobiflight.com/en/download.html)
3. **Go 1.24.7+**
4. **SimConnect.dll** - 从 MSFS SDK 复制

### 安装步骤

```bash
# 1. 克隆仓库
git clone https://github.com/NovaVoyager/Microsoft_Simconnect.git
cd Microsoft_Simconnect

# 2. 复制 SimConnect.dll 到 simconnect/ 目录
# 从以下位置复制:
# C:\Program Files (x86)\Microsoft Flight Simulator\SDK\SimConnect SDK\lib\SimConnect.dll

# 3. 安装 MobiFlight WASM 模块
# 下载并运行 MobiFlight Connector，它会自动安装 WASM 模块
# 或手动下载: https://github.com/MobiFlight/MobiFlight-WASM-Module/releases
```

## 🚀 快速开始

### 测试连接

首先运行测试程序验证设置：

```bash
go run test_connection.go
```

如果看到 "🎉 所有测试通过！"，说明一切正常。

### 基础示例

```go
package main

import (
    "fmt"
    "log"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    // 1. 初始化客户端
    sc, mf, err := simconnect.SetupMobiFlightClient("我的应用")
    if err != nil {
        log.Fatal(err)
    }
    defer sc.Close()

    // 2. 设置飞机参数
    err = mf.SetLVar("XMLVAR_Autopilot_Altitude", 10000.0)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("✓ 自动驾驶高度设置为 10000 英尺")
}
```

### 更多示例

```bash
# 运行基础示例
go run main.go

# 运行高级示例
go run examples/advanced_example.go
```

## 📚 文档

- **[快速入门指南](MOBIFLIGHT_GUIDE.md)** - 完整的使用指南和 API 文档
- **[故障排除](TROUBLESHOOTING.md)** - 常见问题解决方案
- **[调试响应接收](DEBUG_RESPONSES.md)** - 收不到响应数据时的调试步骤
- **[项目架构](CLAUDE.md)** - 代码架构和开发指南

## 🔧 常用操作

### 读取飞机参数

```go
// 发送读取请求
err := mf.GetLVar("XMLVAR_Switch_AP_MSL_Mode")

// 在消息循环中接收响应
response, err := mf.ReceiveResponse()
if err == nil {
    value, _ := simconnect.ParseResponseFloat(response)
    fmt.Printf("当前值: %.2f\n", value)
}
```

### 设置飞机参数

```go
// 设置单个参数
err := mf.SetLVar("XMLVAR_Autopilot_Altitude", 5000.0)
```

### 批量操作

```go
// 批量设置多个参数（提高性能）
batch := simconnect.NewLVarBatch(mf)
batch.Add("XMLVAR_Switch_AP_MSL_Mode", 1.0)
batch.Add("XMLVAR_Autopilot_Altitude", 5000.0)
batch.Add("XMLVAR_Airspeed_Mode", 2.0)
batch.Execute()
```

### 执行 RPN 代码

```go
// 执行自定义 RPN 计算器代码
err := mf.ExecuteCode("100 200 + (>L:MY_VAR)")
```

## 🛠️ 故障排除

### EXCEPTION_OUT_OF_BOUNDS 错误

**原因：** MobiFlight WASM 模块未加载

**解决方案：**
1. 确保 Microsoft Flight Simulator 正在运行
2. 安装 MobiFlight WASM 模块
3. 在 SimConnect Inspector 中验证客户端数据区域存在

详见 [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

### 没有收到响应

**解决方案：**
1. 检查 `ReceiveResponse()` 返回值：`response == nil` 表示没有消息（正常），`err != nil` 表示错误
2. 实现正确的消息循环（参见示例代码）
3. 验证 LVar 名称（使用 MobiFlight Connector）
4. 确保 WASM 模块正在运行
5. 增加等待时间和轮询次数

详见 [DEBUG_RESPONSES.md](DEBUG_RESPONSES.md) 完整调试步骤

## 📖 API 参考

### 核心 API

| 函数 | 说明 |
|------|------|
| `SetupMobiFlightClient(name)` | 一站式初始化 |
| `SetLVar(name, value)` | 设置 LVar |
| `GetLVar(name)` | 读取 LVar |
| `ExecuteCode(rpn)` | 执行 RPN 代码 |
| `ReceiveResponse()` | 接收响应数据 |

### 批量操作 API

| 函数 | 说明 |
|------|------|
| `NewLVarBatch(client)` | 创建批量操作 |
| `Add(name, value)` | 添加操作 |
| `Execute()` | 执行批量操作 |

详见 [MOBIFLIGHT_GUIDE.md](MOBIFLIGHT_GUIDE.md)

## 🎯 常见 LVar 示例

| LVar | 说明 | 范围 |
|------|------|------|
| `XMLVAR_Autopilot_Altitude` | 自动驾驶高度 | 0-50000 |
| `XMLVAR_Switch_AP_MSL_Mode` | 自动驾驶模式 | 0/1 |
| `XMLVAR_Airspeed_Mode` | 空速模式 | 0-3 |

使用 MobiFlight Connector 查找更多 LVar。

## 🔗 相关资源

- [MobiFlight 官网](https://www.mobiflight.com/)
- [SimConnect SDK 文档](https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/SimConnect_SDK.htm)
- [MobiFlight WASM GitHub](https://github.com/MobiFlight/MobiFlight-WASM-Module)
- [RPN 计算器参考](https://docs.flightsimulator.com/html/Additional_Information/Reverse_Polish_Notation.htm)

## 📝 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📧 联系方式

如有问题，请：
1. 查看 [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
2. 提交 GitHub Issue
3. 加入 MobiFlight Discord: https://discord.gg/sqU6eUz