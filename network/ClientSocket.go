package network

import (
	"fmt"
	"gonet/base"
	"gonet/rpc"
	"io"
	"log"
	"net"
)

type IClientSocket interface {
	ISocket
}

type ClientSocket struct {
	Socket
	m_nMaxClients int
	m_nMinClients int
}

//初始化
func (this *ClientSocket) Init(ip string, port int) bool {
	if this.m_nPort == port || this.m_sIP == ip {
		return false
	}

	this.Socket.Init(ip, port)
	this.m_sIP = ip
	this.m_nPort = port
	fmt.Println(ip, port)
	return true
}

//开启连接
func (this *ClientSocket) Start() bool {
	this.m_bShuttingDown = false
	//为空设置本地回环
	if this.m_sIP == "" {
		this.m_sIP = "127.0.0.1"
	}

	if this.Connect() {
		//禁止nagel算法
		this.m_Conn.(*net.TCPConn).SetNoDelay(true)
		//开启一个协程执行任务
		go this.Run()
	}
	//延迟，监听关闭
	//defer ln.Close()
	return true
}

func (this *ClientSocket) Stop() bool {
	//判断是否已经关闭了
	if this.m_bShuttingDown {
		return true
	}
	//设置为关闭状态
	this.m_bShuttingDown = true
	//关闭连接
	this.Close()
	return true
}

//将rpc小子字符串化
func (this *ClientSocket) SendMsg(head rpc.RpcHead, funcName string, params ...interface{}) {
	buff := rpc.Marshal(head, funcName, params...)
	buff = base.SetTcpEnd(buff)
	this.Send(head, buff)
}

//发送数据
func (this *ClientSocket) Send(head rpc.RpcHead, buff []byte) int {
	defer func() {
		if err := recover(); err != nil {
			base.TraceCode(err)
		}
	}()
	//判断连接是否正常
	if this.m_Conn == nil {
		return 0
	} else if len(buff) > base.MAX_PACKET {
		panic("send over base.MAX_PACKET")
	}
	//发送数据
	n, err := this.m_Conn.Write(buff)
	//处理错误
	handleError(err)
	if n > 0 {
		return n
	}
	//this.m_Writer.Flush()
	return 0
}

func (this *ClientSocket) Restart() bool {
	return true
}

func (this *ClientSocket) Connect() bool {
	//如果是已连接状态
	if this.m_nState == SSF_CONNECT {
		return false
	}

	var strRemote = fmt.Sprintf("%s:%d", this.m_sIP, this.m_nPort)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", strRemote)
	if err != nil {
		log.Printf("%v", err)
	}
	//连接
	ln, err1 := net.DialTCP("tcp4", nil, tcpAddr)
	if err1 != nil {
		return false
	}

	//设置一些参数
	this.m_nState = SSF_CONNECT
	this.SetTcpConn(ln)
	fmt.Printf("连接成功，请输入信息！\n")
	this.CallMsg("COMMON_RegisterRequest")
	return true
}

//
func (this *ClientSocket) OnDisconnect() {
}

//网络坏掉了
func (this *ClientSocket) OnNetFail(int) {
	this.Stop()
	this.CallMsg("DISCONNECT", this.m_ClientId)
}

//
func (this *ClientSocket) Run() bool {
	//初始化接受缓冲区
	var buff = make([]byte, this.m_ReceiveBufferSize)
	loop := func() bool {
		defer func() {
			if err := recover(); err != nil {
				base.TraceCode(err)
			}
		}()
		//判断是否被关闭了
		if this.m_bShuttingDown || this.m_Conn == nil {
			return false
		}
		//接收数据
		n, err := this.m_Conn.Read(buff)
		//判断连接是否已关闭
		if err == io.EOF {
			fmt.Printf("远程链接：%s已经关闭！\n", this.m_Conn.RemoteAddr().String())
			this.OnNetFail(0)
			return false
		}
		if err != nil {
			handleError(err)
			this.OnNetFail(0)
			return false
		}
		//处理数据包
		if n > 0 {
			this.ReceivePacket(this.m_ClientId, buff[:n])
		}
		return true
	}

	for {
		if !loop() {
			break
		}
	}
	//关闭
	this.Close()
	return true
}
