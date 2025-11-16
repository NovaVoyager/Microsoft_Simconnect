package main

import (
	"fmt"
	"time"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
	// 1. 一键初始化（使用封装好的辅助函数）
	sc, mf, err := simconnect.SetupMobiFlightClient("APU Reader")
	if err != nil {
		fmt.Printf("❌ 初始化失败: %v\n", err)
		return
	}
	defer sc.Close()

	fmt.Println("✓ 已连接")

	// 2. 清空初始消息
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		sc.GetNextDispatch()
	}

	// 3. 读取 APU 状态
	fmt.Println("\n正在读取 INI_APU_STATE...")
	mf.GetLVar("INI_APU_STATE")

	// 4. 等待响应
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("❌ 超时")
			return

		case <-ticker.C:
			// 接收响应
			response, err := mf.ReceiveResponse()
			if err != nil {
				fmt.Printf("❌ 错误: %v\n", err)
				return
			}

			if response != nil {
				// 成功收到响应
				value, _ := simconnect.ParseResponseFloat(response)
				fmt.Printf("\n✓ APU 状态: %.0f\n", value)

				// 解释状态值
				switch int(value) {
				case 0:
					fmt.Println("  → APU 关闭")
				case 1:
					fmt.Println("  → APU 启动中")
				case 2:
					fmt.Println("  → APU 运行")
				case 3:
					fmt.Println("  → APU 关闭中")
				}
				return
			}
		}
	}
}
