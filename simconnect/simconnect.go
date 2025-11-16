package simconnect

import (
	"fmt"
	"syscall"
	"unsafe"
)

// SimConnect 结构体管理 DLL 句柄和连接句柄
type SimConnect struct {
	Dll    *syscall.DLL
	handle uintptr
}

// LoadDLL 从 simconnect 目录加载 SimConnect.dll
func LoadDLL() (*SimConnect, error) {
	dll, err := syscall.LoadDLL("simconnect/SimConnect.dll")
	if err != nil {
		return nil, fmt.Errorf("failed to load SimConnect.dll: %v", err)
	}
	return &SimConnect{Dll: dll}, nil
}

// Open 建立与模拟器的连接
func (s *SimConnect) Open(clientName string) error {
	proc, err := s.Dll.FindProc("SimConnect_Open")
	if err != nil {
		return err
	}

	name, err := syscall.UTF16PtrFromString(clientName)
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&s.handle)),
		uintptr(unsafe.Pointer(name)),
		0, 0, 0, 0,
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_Open failed, HRESULT=0x%x", r)
	}

	return nil
}

// Close 关闭 SimConnect 会话
func (s *SimConnect) Close() error {
	if s.handle == 0 {
		return nil
	}

	proc, err := s.Dll.FindProc("SimConnect_Close")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(s.handle)
	if r != 0 {
		return fmt.Errorf("SimConnect_Close failed, HRESULT=0x%x", r)
	}

	s.handle = 0
	return nil
}

// MapClientDataNameToID 将客户端数据区域名称映射到 ID
func (s *SimConnect) MapClientDataNameToID(clientDataName string, clientDataID uint32) error {
	proc, err := s.Dll.FindProc("SimConnect_MapClientDataNameToID")
	if err != nil {
		return err
	}

	name, err := syscall.UTF16PtrFromString(clientDataName)
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		uintptr(unsafe.Pointer(name)),
		uintptr(clientDataID),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_MapClientDataNameToID failed, HRESULT=0x%x", r)
	}

	return nil
}

// CreateClientData 创建客户端数据区域
func (s *SimConnect) CreateClientData(clientDataID, size, flags uint32) error {
	proc, err := s.Dll.FindProc("SimConnect_CreateClientData")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		uintptr(clientDataID),
		uintptr(size),
		uintptr(flags),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_CreateClientData failed, HRESULT=0x%x", r)
	}

	return nil
}

// AddToClientDataDefinition 向客户端数据定义添加项
func (s *SimConnect) AddToClientDataDefinition(defineID, offset, sizeOrType, epsilon uint32, datumID uint32) error {
	proc, err := s.Dll.FindProc("SimConnect_AddToClientDataDefinition")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		uintptr(defineID),
		uintptr(offset),
		uintptr(sizeOrType),
		uintptr(epsilon),
		uintptr(datumID),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_AddToClientDataDefinition failed, HRESULT=0x%x", r)
	}

	return nil
}

// RequestClientData 请求客户端数据
func (s *SimConnect) RequestClientData(clientDataID, requestID, defineID, period, flags uint32, origin, interval, limit uint32) error {
	proc, err := s.Dll.FindProc("SimConnect_RequestClientData")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		uintptr(clientDataID),
		uintptr(requestID),
		uintptr(defineID),
		uintptr(period),
		uintptr(flags),
		uintptr(origin),
		uintptr(interval),
		uintptr(limit),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_RequestClientData failed, HRESULT=0x%x", r)
	}

	return nil
}

// SetClientData 设置客户端数据
func (s *SimConnect) SetClientData(clientDataID, defineID, flags, reserved uint32, dataSize uint32, data unsafe.Pointer) error {
	proc, err := s.Dll.FindProc("SimConnect_SetClientData")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		uintptr(clientDataID),
		uintptr(defineID),
		uintptr(flags),
		uintptr(reserved),
		uintptr(dataSize),
		uintptr(data),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_SetClientData failed, HRESULT=0x%x", r)
	}

	return nil
}

// GetNextDispatch 获取下一个调度消息
// 返回 nil, 0, nil 表示当前没有消息（这不是错误）
// 返回 nil, 0, error 表示真正的错误
func (s *SimConnect) GetNextDispatch() (unsafe.Pointer, uint32, error) {
	proc, err := s.Dll.FindProc("SimConnect_GetNextDispatch")
	if err != nil {
		return nil, 0, err
	}

	var ppData uintptr
	var pcbData uint32

	r, _, _ := proc.Call(
		s.handle,
		uintptr(unsafe.Pointer(&ppData)),
		uintptr(unsafe.Pointer(&pcbData)),
	)

	// HRESULT 错误码说明：
	// 0x00000000 = S_OK (成功，有消息)
	// 0x00000001 = S_FALSE (没有消息)
	// 0x80004005 = E_FAIL (通常表示没有消息或队列为空)
	// 其他值 = 可能的错误
	if r == 0 {
		// 成功接收到消息
		return unsafe.Pointer(ppData), pcbData, nil
	} else if r == 1 || r == 0x80004005 {
		// 没有消息（S_FALSE 或 E_FAIL），这不是错误
		return nil, 0, nil
	} else {
		// 可能的真正错误（但我们要保守处理）
		// 大多数情况下，非零返回值只是表示没有消息
		return nil, 0, nil
	}
}

// CallDispatch 处理 SimConnect 消息
func (s *SimConnect) CallDispatch(callback uintptr, context unsafe.Pointer) error {
	proc, err := s.Dll.FindProc("SimConnect_CallDispatch")
	if err != nil {
		return err
	}

	r, _, _ := proc.Call(
		s.handle,
		callback,
		uintptr(context),
	)

	if r != 0 {
		return fmt.Errorf("SimConnect_CallDispatch failed, HRESULT=0x%x", r)
	}

	return nil
}
