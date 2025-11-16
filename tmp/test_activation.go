package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试激活 MobiFlight WASM 模块的不同方法
func main() {
	fmt.Println("=== MobiFlight WASM 激活测试 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("激活测试")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")
	time.Sleep(500 * time.Millisecond)

	// 清空初始消息
	for i := 0; i < 10; i++ {
		sc.GetNextDispatch()
	}

	// 尝试 1: 标准方式
	fmt.Println("=== 尝试 1: 标准 MobiFlight 配置 ===\n")

	sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	sc.MapClientDataNameToID("MobiFlight.Response", 0x00004243)
	sc.CreateClientData(0x00004242, 1024, 0)
	sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)
	sc.AddToClientDataDefinition(0x10001, 0, 1024, 0, 0)

	fmt.Println("发送 'ping' 命令...")
	type Cmd struct {
		Code [1024]byte
	}
	var cmd Cmd
	copy(cmd.Code[:], []byte("MF.Ping"))
	sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))

	time.Sleep(500 * time.Millisecond)

	// 尝试请求数据
	fmt.Println("RequestClientData...")
	err = sc.RequestClientData(0x00004243, 0x20000, 0x10001,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 0, 0, 0, 0)

	if err != nil {
		fmt.Printf("✗ RequestClientData 失败: %v\n", err)
		fmt.Println("  → 这意味着 Response 数据区域不存在！")
		fmt.Println("  → WASM 模块可能未创建数据区域\n")
	} else {
		fmt.Println("✓ RequestClientData 成功")
		fmt.Println("  → Response 数据区域存在\n")

		// 检查是否有响应
		fmt.Println("检查响应...")
		time.Sleep(500 * time.Millisecond)

		foundMsg := false
		for i := 0; i < 10; i++ {
			ppData, _, _ := sc.GetNextDispatch()
			if ppData != nil {
				foundMsg = true
				recv := (*simconnect.SIMCONNECT_RECV)(ppData)
				fmt.Printf("收到消息类型: %d\n", recv.DwID)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					fmt.Println("✓✓ 收到客户端数据消息！")
				}
			}
		}

		if !foundMsg {
			fmt.Println("⚠ 未收到任何消息")
		}
	}

	fmt.Println()

	// 尝试 2: 尝试不同的客户端数据 ID
	fmt.Println("=== 尝试 2: 测试不同的数据区域 ID ===\n")

	alternativeIDs := []struct {
		name       string
		commandID  uint32
		responseID uint32
	}{
		{"MobiFlight 默认", 0x00004242, 0x00004243},
		{"备选 ID 1", 0x00004200, 0x00004201},
		{"备选 ID 2", 0x10000000, 0x10000001},
	}

	for _, alt := range alternativeIDs {
		fmt.Printf("测试: %s (Command=0x%08X, Response=0x%08X)\n", alt.name, alt.commandID, alt.responseID)

		err = sc.RequestClientData(alt.responseID, 0x30000, 0x10001,
			simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE, 0, 0, 0, 0)

		if err != nil {
			fmt.Printf("  ✗ RequestClientData 失败 (数据区域不存在)\n")
		} else {
			fmt.Printf("  ✓ RequestClientData 成功！\n")
			fmt.Printf("  → 可能的正确 ID: 0x%08X\n", alt.responseID)
		}

		fmt.Println()
	}

	// 尝试 3: 发送各种初始化命令
	fmt.Println("=== 尝试 3: 发送可能的初始化命令 ===\n")

	initCommands := []string{
		"MF.Clients.Add.激活测试",
		"MF.Init",
		"MF.Ping",
		"MF.Connect",
		"", // 空命令
	}

	for i, initCmd := range initCommands {
		if initCmd == "" {
			fmt.Printf("命令 %d: (空命令)\n", i+1)
		} else {
			fmt.Printf("命令 %d: %s\n", i+1, initCmd)
		}

		var cmd Cmd
		for j := range cmd.Code {
			cmd.Code[j] = 0
		}
		copy(cmd.Code[:], []byte(initCmd))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ 发送失败: %v\n", err)
		} else {
			fmt.Println("  ✓ 发送成功")

			time.Sleep(300 * time.Millisecond)

			// 尝试请求数据
			err = sc.RequestClientData(0x00004243, uint32(0x40000+i), 0x10001,
				simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE, 0, 0, 0, 0)

			if err == nil {
				fmt.Println("  ✓ 现在可以 RequestClientData 了！")
				fmt.Printf("  → 初始化命令可能是: '%s'\n", initCmd)
			}
		}

		fmt.Println()
	}

	fmt.Println("=== 测试完成 ===\n")

	fmt.Println("结论:")
	fmt.Println("如果所有 RequestClientData 都失败:")
	fmt.Println("  → MobiFlight WASM 模块未创建 Response 数据区域")
	fmt.Println("  → 需要先运行 MobiFlight Connector 来激活模块")
	fmt.Println("  → 或者 WASM 模块可能未正确加载")
	fmt.Println()
	fmt.Println("如果某个 RequestClientData 成功:")
	fmt.Println("  → 找到了正确的数据区域 ID 或初始化方法！")
	fmt.Println("  → 请告诉我哪个配置成功了")
	fmt.Println()
}
