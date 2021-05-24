package network

import (
	"fmt"
	"golang.org/x/net/websocket"
	"gonet/base"
	"gonet/rpc"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

type IWebSocket interface {
	ISocket

	AssignClientId() uint32
	GetClientById(uint32) *WebSocketClient
	LoadClient() *WebSocketClient
	AddClinet(*websocket.Conn, string, int) *WebSocketClient
	DelClinet(*WebSocketClient) bool
	StopClient(uint32)
}

type WebSocket struct {
	Socket
	m_nClientCount int
	m_nMaxClients  int
	m_nMinClients  int
	//id的种子
	m_nIdSeed uint32
	//id与WebSocketClient的映射
	m_ClientList map[uint32]*WebSocketClient
	//读写锁
	m_ClientLocker *sync.RWMutex
	//互斥锁
	m_Lock sync.Mutex
}

//初始化
func (this *WebSocket) Init(ip string, port int) bool {
	this.Socket.Init(ip, port)
	//初始化id映射
	this.m_ClientList = make(map[uint32]*WebSocketClient)
	//初始化锁
	this.m_ClientLocker = &sync.RWMutex{}
	//设置端口和ip
	this.m_sIP = ip
	this.m_nPort = port
	return true
}

//
func (this *WebSocket) Start() bool {
	//未关闭状态
	this.m_bShuttingDown = false

	if this.m_sIP == "" {
		this.m_sIP = "127.0.0.1"
	}
	//
	go func() {
		//监听链接
		var strRemote = fmt.Sprintf("%s:%d", this.m_sIP, this.m_nPort)
		http.Handle("/ws", websocket.Handler(this.wserverRoutine))
		err := http.ListenAndServe(strRemote, nil)
		if err != nil {
			fmt.Errorf("WebSocket ListenAndServe:%v", err)
		}
	}()

	fmt.Printf("WebSocket 启动监听，等待链接！\n")
	//接受连接的状态
	this.m_nState = SSF_ACCEPT
	return true
}

//分配id
func (this *WebSocket) AssignClientId() uint32 {
	return atomic.AddUint32(&this.m_nIdSeed, 1)
}

//根据id返回连接
func (this *WebSocket) GetClientById(id uint32) *WebSocketClient {
	this.m_ClientLocker.RLock()
	client, exist := this.m_ClientList[id]
	this.m_ClientLocker.RUnlock()
	if exist == true {
		return client
	}

	return nil
}

//得到WebSocketClient
func (this *WebSocket) AddClinet(tcpConn *websocket.Conn, addr string, connectType int) *WebSocketClient {
	//
	pClient := this.LoadClient()
	//初始化
	if pClient != nil {
		pClient.Init("", 0)
		pClient.m_pServer = this
		pClient.m_ReceiveBufferSize = this.m_ReceiveBufferSize
		pClient.m_MaxReceiveBufferSize = this.m_MaxReceiveBufferSize
		pClient.m_ClientId = this.AssignClientId()
		pClient.m_sIP = addr
		pClient.SetTcpConn(tcpConn)
		pClient.SetConnectType(connectType)
		this.m_ClientLocker.Lock()
		this.m_ClientList[pClient.m_ClientId] = pClient
		this.m_ClientLocker.Unlock()
		this.m_nClientCount++
		return pClient
	} else {
		log.Printf("%s", "无法创建客户端连接对象")
	}
	return nil
}

//删除连接
func (this *WebSocket) DelClinet(pClient *WebSocketClient) bool {
	this.m_ClientLocker.Lock()
	delete(this.m_ClientList, pClient.m_ClientId)
	this.m_ClientLocker.Unlock()
	return true
}

//关闭id的连接
func (this *WebSocket) StopClient(id uint32) {
	pClinet := this.GetClientById(id)
	if pClinet != nil {
		pClinet.Stop()
	}
}

//生成一个未初始化的连接
func (this *WebSocket) LoadClient() *WebSocketClient {
	s := &WebSocketClient{}
	return s
}

//关闭连接
func (this *WebSocket) Stop() bool {
	if this.m_bShuttingDown {
		return true
	}
	//设置状态
	this.m_bShuttingDown = true
	this.m_nState = SSF_SHUT_DOWN
	return true
}

//发送rpc信息
func (this *WebSocket) Send(head rpc.RpcHead, buff []byte) int {
	pClient := this.GetClientById(head.SocketId)
	if pClient != nil {
		pClient.Send(head, base.SetTcpEnd(buff))
	}
	return 0
}

//
func (this *WebSocket) SendMsg(head rpc.RpcHead, funcName string, params ...interface{}) {
	pClient := this.GetClientById(head.SocketId)
	if pClient != nil {
		pClient.Send(head, base.SetTcpEnd(rpc.Marshal(head, funcName, params...)))
	}
}

func (this *WebSocket) Restart() bool {
	return true
}
func (this *WebSocket) Connect() bool {
	return true
}
func (this *WebSocket) Disconnect(bool) bool {
	return true
}

func (this *WebSocket) OnNetFail(int) {
}

func (this *WebSocket) Close() {
	this.Clear()
}

func (this *WebSocket) wserverRoutine(conn *websocket.Conn) {
	fmt.Printf("客户端：%s已连接！\n", conn.RemoteAddr().String())
	//
	this.handleConn(conn, conn.RemoteAddr().String())
}

//处理连接
func (this *WebSocket) handleConn(tcpConn *websocket.Conn, addr string) bool {
	if tcpConn == nil {
		return false
	}

	tcpConn.PayloadType = websocket.BinaryFrame
	//封装成一个WebSocketClient
	pClient := this.AddClinet(tcpConn, addr, this.m_nConnectType)
	if pClient == nil {
		return false
	}

	pClient.Start()
	return true
}
