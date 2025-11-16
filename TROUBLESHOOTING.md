# MobiFlight WASM 模块故障排除指南

## 常见错误及解决方案

### 1. EXCEPTION_OUT_OF_BOUNDS

**错误信息：**
```
SetClientData: clientDataID=16962, DefineID=65536, Flags=0, dwReserved=0, cbUnitSize=1024
EXCEPTION_OUT_OF_BOUNDS
```

**原因：**
- 尝试访问未创建的客户端数据区域
- MobiFlight WASM 模块未运行或未正确加载

**解决方案：**

#### 步骤 1: 检查 MobiFlight WASM 模块是否安装

1. 导航到你的 MSFS 社区文件夹：
   ```
   %APPDATA%\Microsoft Flight Simulator\Packages\Community
   或
   %LOCALAPPDATA%\Packages\Microsoft.FlightSimulator_8wekyb3d8bbwe\LocalCache\Packages\Community
   ```

2. 确认存在 `mobiflight-event-module` 文件夹

3. 文件夹结构应该类似：
   ```
   mobiflight-event-module/
   ├── layout.json
   ├── manifest.json
   └── modules/
       └── MobiFlightWasmModule.wasm
   ```

#### 步骤 2: 安装 MobiFlight WASM 模块

如果未安装，请按以下步骤操作：

1. **下载 MobiFlight Connector**
   - 访问: https://www.mobiflight.com/en/download.html
   - 下载最新版本的 MobiFlight Connector

2. **安装 WASM 模块**
   - 运行 MobiFlight Connector
   - 连接到 MSFS
   - MobiFlight Connector 会自动安装 WASM 模块

3. **手动安装（如果自动安装失败）**
   - 从 GitHub 下载: https://github.com/MobiFlight/MobiFlight-WASM-Module/releases
   - 解压到社区文件夹
   - 重启模拟器

#### 步骤 3: 验证 WASM 模块已加载

在 MSFS 中：

1. 进入主菜单
2. 选择 **选项** > **常规** > **开发者模式** (打开)
3. 打开 **行为调试窗口** (Behavior Debug Window)
4. 查找 `MobiFlight` 相关的 WASM 模块加载信息

或使用 SimConnect Inspector：

1. 启动 MSFS
2. 运行 SimConnect Inspector
3. 在 "Client Data Area" 标签中查找：
   - `MobiFlight.Command`
   - `MobiFlight.Response`
   - `MobiFlight.LVars`

如果看到这些条目，说明 WASM 模块已正确加载。

#### 步骤 4: 代码修复（已包含在最新版本中）

确保你的代码在发送命令前创建了客户端数据区域：

```go
// 这已经在最新的 initialize() 函数中处理了
_ = m.sc.CreateClientData(MOBIFLIGHT_CLIENT_DATA_ID_COMMAND, MOBIFLIGHT_MESSAGE_SIZE, 0)
```

---

### 2. EXCEPTION_ILLEGAL_OPERATION

**错误信息：**
```
SetClientData: clientDataID=16962, DefineID=65536, Flags=0, dwReserved=0, cbUnitSize=1024
EXCEPTION_ILLEGAL_OPERATION
```

**原因：**
- 尝试在未定义的数据结构上操作
- 数据区域的大小或偏移量不正确
- WASM 模块未运行

**解决方案：**

1. **检查数据区域定义顺序：**
   ```go
   // 正确的顺序：
   // 1. 映射名称到 ID
   MapClientDataNameToID(...)

   // 2. 创建数据区域（如果需要）
   CreateClientData(...)

   // 3. 定义数据结构
   AddToClientDataDefinition(...)

   // 4. 然后才能读写数据
   SetClientData(...) 或 RequestClientData(...)
   ```

2. **确保 WASM 模块正在运行**（参见上一节）

3. **检查数据大小匹配：**
   ```go
   // 确保所有地方使用相同的大小常量
   const MOBIFLIGHT_MESSAGE_SIZE = 1024
   ```

---

### 3. 没有收到响应数据

**症状：**
- `GetLVar()` 调用成功，但 `ReceiveResponse()` 总是返回错误
- 超时未收到任何数据

**原因：**
- 消息循环实现不正确
- LVar 名称错误
- WASM 模块未响应

**解决方案：**

#### 方案 1: 正确实现消息循环

```go
// 创建一个 goroutine 持续处理消息
go func() {
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            response, err := mf.ReceiveResponse()
            if err != nil {
                continue // 没有消息，继续等待
            }

            // 处理响应
            value, _ := simconnect.ParseResponseFloat(response)
            fmt.Printf("收到值: %.2f\n", value)
        }
    }
}()
```

#### 方案 2: 验证 LVar 名称

使用 MobiFlight Connector 验证 LVar 名称：

1. 打开 MobiFlight Connector
2. 连接到 MSFS
3. 在 "Variables" 窗口中搜索你的 LVar
4. 确认名称拼写正确

#### 方案 3: 检查 WASM 模块日志

1. 启用开发者模式
2. 查看控制台输出
3. 检查是否有 WASM 模块错误信息

---

### 4. SimConnect_Open 失败

**错误信息：**
```
SimConnect_Open failed, HRESULT=0x887A0001
```

**原因：**
- MSFS 未运行
- SimConnect.dll 版本不匹配

**解决方案：**

1. **确保 MSFS 正在运行**
   - 启动 Microsoft Flight Simulator
   - 进入主菜单或载入飞行

2. **使用正确版本的 SimConnect.dll**
   - 从 MSFS 安装目录复制 SimConnect.dll：
     ```
     C:\Program Files (x86)\Microsoft Flight Simulator\SDK\SimConnect SDK\lib\SimConnect.dll
     ```
   - 复制到你的项目 `simconnect/` 目录

3. **检查防火墙设置**
   - 允许你的应用程序通过防火墙
   - 允许 MSFS 通过防火墙

---

### 5. LVar 值不更新

**症状：**
- `SetLVar()` 调用成功，但飞机参数没有变化

**原因：**
- LVar 名称错误
- LVar 是只读的
- 值超出有效范围
- 飞机不使用该 LVar

**解决方案：**

1. **使用 MobiFlight Connector 测试：**
   - 先在 MobiFlight Connector 中测试 LVar
   - 确认可以成功读写
   - 然后在你的代码中使用相同的名称

2. **检查值的范围：**
   ```go
   // 某些 LVar 只接受特定范围的值
   // 例如：开关通常是 0 或 1
   mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 1.0) // ✓ 正确
   mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 5.0) // ✗ 可能无效
   ```

3. **查阅飞机文档：**
   - 每个飞机的 LVar 都不同
   - 查看飞机开发者提供的文档
   - 或使用 FSUIPC 的 LVar 监控功能

---

## 调试技巧

### 1. 启用 SimConnect 日志

在 MSFS 安装目录创建 `SimConnect.xml`：

```xml
<?xml version="1.0" encoding="utf-8"?>
<SimBase.Document Type="SimConnect" version="1,0">
  <SimConnect.LogFile>
    <![CDATA[
      <Scope>ALL</Scope>
      <Level>VERBOSE</Level>
      <Destinations>
        <File>%TEMP%\SimConnect_%03u.log</File>
      </Destinations>
    ]]>
  </SimConnect.LogFile>
</SimBase.Document>
```

日志文件位置：`%TEMP%\SimConnect_001.log`

### 2. 使用 SimConnect Inspector

下载并运行 SimConnect Inspector：
- 查看客户端数据区域
- 监控 SimConnect 事件
- 查看错误代码

### 3. 添加详细日志到你的代码

```go
log.Printf("正在初始化 MobiFlight 客户端...")
sc, mf, err := simconnect.SetupMobiFlightClient("我的应用")
if err != nil {
    log.Printf("初始化失败: %v", err)
    return err
}
log.Printf("✓ 初始化成功")

log.Printf("正在设置 LVar: %s = %.2f", varName, value)
err = mf.SetLVar(varName, value)
if err != nil {
    log.Printf("设置 LVar 失败: %v", err)
    return err
}
log.Printf("✓ LVar 设置成功")
```

### 4. 使用最小测试程序

创建一个最小的测试程序来隔离问题：

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
    fmt.Println("=== 最小测试程序 ===")

    // 1. 连接测试
    sc, mf, err := simconnect.SetupMobiFlightClient("测试")
    if err != nil {
        log.Fatal("连接失败:", err)
    }
    defer sc.Close()
    fmt.Println("✓ 连接成功")

    // 2. 设置测试
    err = mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 1.0)
    if err != nil {
        log.Fatal("设置失败:", err)
    }
    fmt.Println("✓ 设置成功")

    // 3. 读取测试
    err = mf.GetLVar("XMLVAR_Switch_AP_MSL_Mode")
    if err != nil {
        log.Fatal("读取请求失败:", err)
    }
    fmt.Println("✓ 读取请求成功")

    // 4. 等待响应
    time.Sleep(500 * time.Millisecond)
    response, err := mf.ReceiveResponse()
    if err != nil {
        fmt.Println("⚠ 未收到响应（这可能是正常的）")
    } else {
        value, _ := simconnect.ParseResponseFloat(response)
        fmt.Printf("✓ 收到响应: %.2f\n", value)
    }
}
```

---

## 常见 HRESULT 错误码

| 错误码 | 含义 | 解决方案 |
|--------|------|----------|
| 0x00000000 | 成功 | - |
| 0x887A0001 | 模拟器未运行 | 启动 MSFS |
| 0x887A0005 | 连接超时 | 检查网络和防火墙 |
| 0x80004005 | 一般错误 | 检查参数和调用顺序 |
| 0x80070057 | 无效参数 | 检查传递的参数值 |

---

## 检查清单

在报告问题前，请确认：

- [ ] Microsoft Flight Simulator 正在运行
- [ ] MobiFlight WASM 模块已安装并加载
- [ ] SimConnect.dll 版本正确
- [ ] 防火墙允许应用程序连接
- [ ] LVar 名称正确（使用 MobiFlight Connector 验证）
- [ ] 使用最新版本的代码
- [ ] 查看了 SimConnect 日志文件
- [ ] 尝试了最小测试程序

---

## 获取帮助

如果问题仍未解决：

1. **查看 MobiFlight 官方文档：**
   - https://www.mobiflight.com/
   - https://github.com/MobiFlight/MobiFlight-Connector/wiki

2. **检查 GitHub Issues：**
   - 搜索类似问题
   - 创建新的 issue 并提供：
     - 错误信息的完整截图
     - SimConnect 日志文件
     - 你使用的 MSFS 版本
     - 你的代码示例

3. **MobiFlight 社区：**
   - Discord: https://discord.gg/sqU6eUz
   - 论坛: https://forums.flightsimulator.com/

---

## 更新日志

### 2025-01-15
- 添加 `CreateClientData` 调用到初始化函数
- 修复 `GetLVar` 命令格式（使用 RPN 格式）
- 添加 LVar 名称前缀处理
- 更新文档和示例

### 初始版本
- 基础 MobiFlight WASM 模块封装
- SimConnect 基础功能
