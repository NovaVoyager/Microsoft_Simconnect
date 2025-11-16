# 调试响应接收问题

如果您运行测试程序后仍然收不到响应数据，请按照以下步骤进行调试。

## 问题概述

您遇到的问题是：
- `SetLVar()` 调用成功（没有错误）
- `GetLVar()` 请求发送成功（没有错误）
- 但是 `ReceiveResponse()` 总是返回 `nil, nil`（表示没有消息）

## 调试步骤

### 步骤 1: 验证 MobiFlight WASM 模块已加载

#### 方法 1: 使用 MobiFlight Connector

1. 下载并运行 [MobiFlight Connector](https://www.mobiflight.com/en/download.html)
2. 启动 Microsoft Flight Simulator
3. 在 MobiFlight Connector 中点击 "Connect"
4. 查看连接状态：
   - 如果显示 "Connected" 并且能看到 WASM 状态，说明模块已正确加载
   - 如果显示错误或 WASM 状态为红色，说明模块未加载

#### 方法 2: 使用 SimConnect Inspector

1. 从 MSFS SDK 运行 SimConnect Inspector
2. 在 "Client Data Areas" 标签查找：
   ```
   MobiFlight.Command
   MobiFlight.Response
   MobiFlight.LVars
   ```
3. 如果看到这些条目，说明 WASM 模块已加载
4. 如果没有看到，说明 WASM 模块未加载

#### 方法 3: 检查社区文件夹

确认文件夹存在：
```
%APPDATA%\Microsoft Flight Simulator\Packages\Community\mobiflight-event-module\
```

或者：
```
%LOCALAPPDATA%\Packages\Microsoft.FlightSimulator_8wekyb3d8bbwe\LocalCache\Packages\Community\mobiflight-event-module\
```

### 步骤 2: 添加调试输出

修改你的代码以添加详细的调试信息：

```go
package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
	// 初始化
	sc, mf, err := simconnect.SetupMobiFlightClient("调试程序")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("✓ 连接成功")

	// 设置 LVar
	fmt.Println("\n发送 SetLVar 命令...")
	err = mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 1.0)
	if err != nil {
		log.Fatalf("SetLVar 失败: %v", err)
	}
	fmt.Println("✓ SetLVar 命令已发送")

	time.Sleep(500 * time.Millisecond)

	// 请求读取
	fmt.Println("\n发送 GetLVar 请求...")
	err = mf.GetLVar("XMLVAR_Switch_AP_MSL_Mode")
	if err != nil {
		log.Fatalf("GetLVar 失败: %v", err)
	}
	fmt.Println("✓ GetLVar 请求已发送")

	// 轮询消息
	fmt.Println("\n开始轮询消息...")
	for i := 0; i < 100; i++ {
		response, err := mf.ReceiveResponse()

		if err != nil {
			fmt.Printf("[%d] 错误: %v\n", i, err)
			break
		}

		if response != nil {
			fmt.Printf("[%d] ✓ 收到响应！\n", i)
			value, _ := simconnect.ParseResponseFloat(response)
			fmt.Printf("值: %.6f\n", value)
			break
		} else {
			if i%10 == 0 {
				fmt.Printf("[%d] 等待响应...\n", i)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}
```

### 步骤 3: 检查 SimConnect 消息

添加原始消息调试：

```go
// 在接收循环中添加
func debugMessages(sc *simconnect.SimConnect) {
	for i := 0; i < 50; i++ {
		ppData, pcbData, err := sc.GetNextDispatch()

		if err != nil {
			fmt.Printf("GetNextDispatch 错误: %v\n", err)
			break
		}

		if ppData == nil {
			fmt.Printf("[%d] 没有消息\n", i)
		} else {
			recv := (*simconnect.SIMCONNECT_RECV)(ppData)
			fmt.Printf("[%d] 收到消息: ID=%d, Size=%d, Version=%d\n",
				i, recv.DwID, recv.DwSize, recv.DwVersion)

			// 如果是客户端数据消息
			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
				fmt.Printf("    ClientData: RequestID=%d, DefineID=%d\n",
					clientData.RequestID, clientData.DefineID)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}
```

### 步骤 4: 测试不同的 LVar

某些 LVar 可能只在特定飞机中存在。尝试使用通用的 LVar：

```go
// 尝试这些更通用的变量
testVars := []string{
	"L:XMLVAR_Switch_AP_MSL_Mode",
	"L:XMLVAR_Autopilot_Altitude",
	"L:XMLVAR_BARO1_Mode",
}

for _, varName := range testVars {
	fmt.Printf("\n测试 %s...\n", varName)
	err := mf.GetLVar(varName)
	if err != nil {
		fmt.Printf("  请求失败: %v\n", err)
		continue
	}

	time.Sleep(500 * time.Millisecond)

	response, err := mf.ReceiveResponse()
	if err != nil {
		fmt.Printf("  接收错误: %v\n", err)
	} else if response != nil {
		value, _ := simconnect.ParseResponseFloat(response)
		fmt.Printf("  ✓ 值: %.2f\n", value)
	} else {
		fmt.Printf("  没有响应\n")
	}
}
```

### 步骤 5: 使用 MobiFlight Connector 验证

在运行你的 Go 程序之前，先用 MobiFlight Connector 测试相同的 LVar：

1. 打开 MobiFlight Connector
2. 连接到 MSFS
3. 进入飞机（确保飞机已完全加载）
4. 在 MobiFlight Connector 中添加一个输入：
   - Type: "Analog Input"
   - Input Type: "LVar"
   - LVar Name: "XMLVAR_Switch_AP_MSL_Mode"
5. 观察值是否更新
6. 如果 MobiFlight Connector 也收不到值，说明：
   - LVar 名称错误，或
   - 当前飞机不支持这个 LVar

### 步骤 6: 检查数据区域设置

验证客户端数据区域配置：

```go
func verifyClientDataAreas(sc *simconnect.SimConnect) error {
	// 打印配置信息
	fmt.Println("验证客户端数据区域配置:")
	fmt.Printf("  Command Area ID: 0x%X\n", simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND)
	fmt.Printf("  Response Area ID: 0x%X\n", simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE)
	fmt.Printf("  Message Size: %d\n", simconnect.MOBIFLIGHT_MESSAGE_SIZE)

	return nil
}
```

### 步骤 7: 启用 SimConnect 详细日志

创建 `SimConnect.xml` 在 MSFS 安装目录：

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

然后检查日志文件：`%TEMP%\SimConnect_001.log`

查找：
- `MapClientDataNameToID` 调用
- `RequestClientData` 调用
- `SetClientData` 调用
- 任何错误或警告

## 常见问题和解决方案

### 问题 1: WASM 模块未加载

**症状：** SimConnect Inspector 中看不到 MobiFlight 数据区域

**解决方案：**
1. 重新安装 MobiFlight WASM 模块
2. 确保社区文件夹路径正确
3. 重启 MSFS

### 问题 2: 飞机不支持 LVar

**症状：** MobiFlight Connector 也无法读取该 LVar

**解决方案：**
1. 使用 MobiFlight Connector 查看当前飞机支持的 LVar 列表
2. 尝试使用其他 LVar
3. 某些飞机（如默认飞机）可能不使用 LVar 系统

### 问题 3: 响应数据格式错误

**症状：** 收到响应但解析失败

**解决方案：**
```go
// 打印原始响应数据
response, err := mf.ReceiveResponse()
if response != nil {
	fmt.Printf("原始响应 (前 100 字节): %v\n", response.Data[:100])
	str := simconnect.ParseResponseString(response)
	fmt.Printf("字符串形式: '%s'\n", str)
}
```

### 问题 4: 时间问题

**症状：** 有时收到响应，有时收不到

**解决方案：**
1. 增加等待时间：`time.Sleep(1 * time.Second)`
2. 增加轮询次数
3. 确保飞机已完全加载

## 替代方法：使用标准 SimVar

如果 LVar 方式不工作，可以尝试使用标准的 SimConnect SimVar：

```go
// 使用标准 SimVar（不需要 MobiFlight WASM）
func useStandardSimVars(sc *simconnect.SimConnect) error {
	// 定义数据结构
	defineID := uint32(1)
	requestID := uint32(1)

	// 添加到数据定义
	// 注意：这需要额外实现 AddToDataDefinition 函数
	// 示例：读取高度
	// sc.AddToDataDefinition(defineID, "Plane Altitude", "feet", ...)

	// 这是一个更可靠的方法，但需要更多代码
	return nil
}
```

## 如果仍然无法解决

1. **确认环境：**
   - MSFS 版本？（2020 还是 2024？）
   - 使用的飞机？
   - MobiFlight WASM 模块版本？

2. **收集日志：**
   - SimConnect 日志
   - MobiFlight Connector 日志
   - 你的程序输出

3. **最小复现：**
   ```go
   // 创建最小的复现代码
   package main

   import (
       "fmt"
       "log"
       "time"
       "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
   )

   func main() {
       sc, mf, _ := simconnect.SetupMobiFlightClient("最小测试")
       defer sc.Close()

       mf.SetLVar("XMLVAR_Switch_AP_MSL_Mode", 1.0)
       time.Sleep(1 * time.Second)

       for i := 0; i < 10; i++ {
           resp, err := mf.ReceiveResponse()
           fmt.Printf("%d: resp=%v, err=%v\n", i, resp != nil, err)
           time.Sleep(200 * time.Millisecond)
       }
   }
   ```

4. **提交 Issue：**
   - 包含完整的错误信息
   - 包含环境信息
   - 包含最小复现代码
   - 包含 SimConnect 日志（如果有）

## 下一步

如果按照以上所有步骤仍然无法收到响应，可能的原因：

1. **WASM 模块版本问题** - 尝试不同版本的 MobiFlight WASM 模块
2. **MSFS 版本兼容性** - 某些版本可能有不同的行为
3. **代码实现问题** - 可能需要调整消息处理机制

建议：先用 MobiFlight Connector 验证一切正常，然后再尝试自己的代码。
