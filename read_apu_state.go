package main

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
	fmt.Println("正在连接到 Microsoft Flight Simulator...")

	// 1. 加载 SimConnect DLL 并建立连接
	sc, err := simconnect.LoadDLL()
	if err != nil {
		fmt.Printf("❌ 加载 SimConnect DLL 失败: %v\n", err)
		return
	}

	err = sc.Open("APU State Reader")
	if err != nil {
		fmt.Printf("❌ 连接 SimConnect 失败: %v\n", err)
		return
	}
	defer sc.Close()

	fmt.Println("✓ 已连接到 SimConnect")

	// 2. 初始化 MobiFlight 客户端
	mf, err := simconnect.NewMobiFlightClient(sc)
	if err != nil {
		fmt.Printf("❌ 初始化 MobiFlight 客户端失败: %v\n", err)
		return
	}

	fmt.Println("✓ MobiFlight 客户端已初始化")

	// 3. 清空初始消息（连接时会收到一些初始化消息）
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		sc.GetNextDispatch()
	}

	// 4. 发送读取 LVar 的命令
	fmt.Println("\n正在读取 INI_APU_STATE...")
	err = mf.GetLVar("INI_APU_STATE")
	if err != nil {
		fmt.Printf("❌ 发送读取命令失败: %v\n", err)
		return
	}

	fmt.Println("✓ 读取命令已发送")
	fmt.Println("等待响应...")

	// 5. 接收响应（使用超时机制）
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("\n❌ 超时：未收到响应")
			fmt.Println("\n可能的原因：")
			fmt.Println("1. MobiFlight WASM 模块未正确加载")
			fmt.Println("2. 当前飞机没有 INI_APU_STATE 这个 LVar")
			fmt.Println("3. 需要在飞行中（而不是主菜单）")
			return

		case <-ticker.C:
			// 尝试接收消息
			ppData, _, _ := sc.GetNextDispatch()
			if ppData == nil {
				continue
			}

			// 检查是否为客户端数据消息
			recv := (*simconnect.SIMCONNECT_RECV)(ppData)
			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

				// 检查是否为 MobiFlight 响应
				if clientData.RequestID == simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE {
					fmt.Println("\n✓✓ 收到响应！")

					// 解析响应数据
					dataPtr := unsafe.Pointer(uintptr(ppData) +
						unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
					response := (*simconnect.MobiFlightResponse)(dataPtr)

					// 获取原始字符串
					str := simconnect.ParseResponseString(response)
					fmt.Printf("原始数据: '%s'\n", str)

					// 尝试解析为浮点数
					value, err := simconnect.ParseResponseFloat(response)
					if err != nil {
						fmt.Printf("⚠ 无法解析为数字: %v\n", err)
					} else {
						fmt.Printf("APU 状态值: %.0f\n", value)

						// 解释 APU 状态值的含义
						fmt.Println("\nAPU 状态说明:")
						switch int(value) {
						case 0:
							fmt.Println("0 = APU 关闭")
						case 1:
							fmt.Println("1 = APU 启动中")
						case 2:
							fmt.Println("2 = APU 运行")
						case 3:
							fmt.Println("3 = APU 关闭中")
						default:
							fmt.Printf("%.0f = 未知状态\n", value)
						}
					}

					return
				}
			}
		}
	}
}
