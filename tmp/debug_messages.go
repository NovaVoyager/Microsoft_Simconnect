package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 这个程序用于调试 SimConnect 消息接收
func main() {
	fmt.Println("=================================")
	fmt.Println("SimConnect 消息调试工具")
	fmt.Println("=================================\n")

	// 初始化
	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatalf("加载 DLL 失败: %v", err)
	}

	err = sc.Open("x")
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")

	// 初始化 MobiFlight 客户端
	mf, err := simconnect.NewMobiFlightClient(sc)
	if err != nil {
		log.Fatalf("MobiFlight 初始化失败: %v", err)
	}

	fmt.Println("✓ MobiFlight 客户端初始化成功\n")

	// 测试 1: 发送设置命令
	fmt.Println("=== 测试 1: 设置 LVar ===")
	testVar := "XMLVAR_Switch_AP_MSL_Mode"
	fmt.Printf("设置 L:%s = 1.0\n", testVar)

	err = mf.SetLVar(testVar, 1.0)
	if err != nil {
		fmt.Printf("✗ 设置失败: %v\n\n", err)
	} else {
		fmt.Println("✓ 设置命令已发送\n")
	}

	time.Sleep(500 * time.Millisecond)

	// 测试 2: 发送读取请求
	fmt.Println("=== 测试 2: 读取 LVar ===")
	fmt.Printf("请求读取 L:%s\n", testVar)

	err = mf.GetLVar(testVar)
	if err != nil {
		fmt.Printf("✗ 读取请求失败: %v\n\n", err)
	} else {
		fmt.Println("✓ 读取请求已发送\n")
	}

	// 测试 3: 调试原始消息
	fmt.Println("=== 测试 3: 监听所有 SimConnect 消息 ===")
	fmt.Println("监听 10 秒，查看收到的所有消息...\n")

	messageCount := 0
	clientDataCount := 0
	responseCount := 0

	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			goto Summary

		case <-ticker.C:
			ppData, pcbData, err := sc.GetNextDispatch()

			if err != nil {
				fmt.Printf("获取消息时出错: %v\n", err)
				continue
			}

			if ppData == nil {
				// 没有消息
				continue
			}

			// 收到消息
			messageCount++

			recv := (*simconnect.SIMCONNECT_RECV)(ppData)
			elapsed := time.Since(startTime).Seconds()

			fmt.Printf("[%.2fs] 消息 #%d:\n", elapsed, messageCount)
			fmt.Printf("  消息类型 ID: %d (0x%x)\n", recv.DwID, recv.DwID)
			fmt.Printf("  消息大小: %d 字节\n", recv.DwSize)
			fmt.Printf("  消息版本: %d\n", recv.DwVersion)
			fmt.Printf("  数据大小: %d 字节\n", pcbData)

			// 检查是否是客户端数据消息
			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				clientDataCount++
				clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

				fmt.Printf("  ✓ 这是客户端数据消息！\n")
				fmt.Printf("    Request ID: %d (期望: %d)\n",
					clientData.RequestID, simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE)
				fmt.Printf("    Object ID: %d\n", clientData.ObjectID)
				fmt.Printf("    Define ID: %d (期望: %d)\n",
					clientData.DefineID, simconnect.MOBIFLIGHT_DEFINE_ID_RESPONSE)
				fmt.Printf("    Flags: %d\n", clientData.Flags)
				fmt.Printf("    Entry: %d of %d\n", clientData.EntryNumber, clientData.OutOf)
				fmt.Printf("    Define Count: %d\n", clientData.DefineCount)

				// 检查是否是我们期望的响应
				if clientData.RequestID == simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE {
					responseCount++
					fmt.Println("    ✓✓ 这是 MobiFlight 响应消息！")

					// 读取数据
					dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
					response := (*simconnect.MobiFlightResponse)(dataPtr)

					// 显示原始数据
					fmt.Printf("    原始数据 (前 100 字节): %v\n", response.Data[:100])

					// 尝试解析
					str := simconnect.ParseResponseString(response)
					fmt.Printf("    字符串数据: '%s'\n", str)

					if str != "" {
						value, err := simconnect.ParseResponseFloat(response)
						if err == nil {
							fmt.Printf("    ✓✓✓ 解析成功！值: %.6f\n", value)
						} else {
							fmt.Printf("    解析为浮点数失败: %v\n", err)
						}
					}
				}
			} else {
				// 其他类型的消息
				messageTypeName := getMessageTypeName(recv.DwID)
				fmt.Printf("  消息类型: %s\n", messageTypeName)
			}

			fmt.Println()
		}
	}

Summary:
	fmt.Println("\n=================================")
	fmt.Println("统计信息")
	fmt.Println("=================================")
	fmt.Printf("总消息数: %d\n", messageCount)
	fmt.Printf("客户端数据消息数: %d\n", clientDataCount)
	fmt.Printf("MobiFlight 响应消息数: %d\n", responseCount)

	if responseCount > 0 {
		fmt.Println("\n✓✓✓ 成功！收到了 MobiFlight 响应！")
		fmt.Println("您的设置是正确的。")
	} else if clientDataCount > 0 {
		fmt.Println("\n⚠ 收到了客户端数据消息，但不是 MobiFlight 响应。")
		fmt.Println("可能的原因:")
		fmt.Println("  1. Request ID 或 Define ID 不匹配")
		fmt.Println("  2. MobiFlight WASM 模块使用了不同的 ID")
		fmt.Println("建议: 查看上面的 Request ID 和 Define ID，可能需要调整常量")
	} else if messageCount > 0 {
		fmt.Println("\n⚠ 收到了 SimConnect 消息，但没有客户端数据消息。")
		fmt.Println("可能的原因:")
		fmt.Println("  1. MobiFlight WASM 模块未运行")
		fmt.Println("  2. 客户端数据区域未正确设置")
		fmt.Println("  3. RequestClientData 调用失败")
		fmt.Println("建议: 使用 MobiFlight Connector 验证 WASM 模块已加载")
	} else {
		fmt.Println("\n✗ 完全没有收到任何消息。")
		fmt.Println("可能的原因:")
		fmt.Println("  1. SimConnect 连接有问题")
		fmt.Println("  2. 模拟器未运行或未载入飞行")
		fmt.Println("  3. GetNextDispatch 调用有问题")
		fmt.Println("建议:")
		fmt.Println("  - 确保进入了飞行（不只是主菜单）")
		fmt.Println("  - 尝试在模拟器中执行一些操作")
		fmt.Println("  - 检查 SimConnect.log")
	}

	fmt.Println()
}

// 获取消息类型名称
func getMessageTypeName(id uint32) string {
	switch id {
	case simconnect.SIMCONNECT_RECV_ID_NULL:
		return "NULL"
	case simconnect.SIMCONNECT_RECV_ID_EXCEPTION:
		return "EXCEPTION"
	case simconnect.SIMCONNECT_RECV_ID_OPEN:
		return "OPEN"
	case simconnect.SIMCONNECT_RECV_ID_QUIT:
		return "QUIT"
	case simconnect.SIMCONNECT_RECV_ID_EVENT:
		return "EVENT"
	case simconnect.SIMCONNECT_RECV_ID_EVENT_OBJECT_ADDREMOVE:
		return "EVENT_OBJECT_ADDREMOVE"
	case simconnect.SIMCONNECT_RECV_ID_EVENT_FILENAME:
		return "EVENT_FILENAME"
	case simconnect.SIMCONNECT_RECV_ID_EVENT_FRAME:
		return "EVENT_FRAME"
	case simconnect.SIMCONNECT_RECV_ID_SIMOBJECT_DATA:
		return "SIMOBJECT_DATA"
	case simconnect.SIMCONNECT_RECV_ID_SIMOBJECT_DATA_BYTYPE:
		return "SIMOBJECT_DATA_BYTYPE"
	case simconnect.SIMCONNECT_RECV_ID_CLOUD_STATE:
		return "CLOUD_STATE"
	case simconnect.SIMCONNECT_RECV_ID_ASSIGNED_OBJECT_ID:
		return "ASSIGNED_OBJECT_ID"
	case simconnect.SIMCONNECT_RECV_ID_RESERVED_KEY:
		return "RESERVED_KEY"
	case simconnect.SIMCONNECT_RECV_ID_CUSTOM_ACTION:
		return "CUSTOM_ACTION"
	case simconnect.SIMCONNECT_RECV_ID_SYSTEM_STATE:
		return "SYSTEM_STATE"
	case simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA:
		return "CLIENT_DATA"
	case simconnect.SIMCONNECT_RECV_ID_EVENT_WEATHER_MODE:
		return "EVENT_WEATHER_MODE"
	case simconnect.SIMCONNECT_RECV_ID_AIRPORT_LIST:
		return "AIRPORT_LIST"
	case simconnect.SIMCONNECT_RECV_ID_VOR_LIST:
		return "VOR_LIST"
	case simconnect.SIMCONNECT_RECV_ID_NDB_LIST:
		return "NDB_LIST"
	case simconnect.SIMCONNECT_RECV_ID_WAYPOINT_LIST:
		return "WAYPOINT_LIST"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", id)
	}
}
