package network

import (
	"fmt"
	"gonet/base"
	"gonet/rpc"
	"hash/crc32"
	"io"
	"log"
	"net"
)

const (
	IDLE_TIMEOUT    = iota
	CONNECT_TIMEOUT = iota
	CONNECT_TYPE    = iota
)

var (
	DISCONNECTINT = crc32.ChecksumIEEE([]byte("DISCONNECT"))
)

type IServerSocketClient interface {
	ISocket
}

type ServerSocketClient struct {
	Socket
	//关联的ServerSocket
	m_pServer *ServerSocket
	//
	m_SendChan chan []byte //对外缓冲队列
}

//处理错误
func handleError(err error) {
	if err == nil {
		return
	}
	log.Printf("错误：%s\n", err.Error())
}

//初始化
func (this *ServerSocketClient) Init(ip string, port int) bool {
	//如果已连接
	if this.m_nConnectType == CLIENT_CONNECT {
		this.m_SendChan = make(chan []byte, MAX_SEND_CHAN)
	}
	//设置端口和ip
	this.Socket.Init(ip, port)
	return true
}

//
func (this *ServerSocketClient) Start() bool {
	//如果不处于关闭状态
	if this.m_nState != SSF_SHUT_DOWN {
		return false
	}
	if this.m_pServer == nil {
		return false
	}
	//
	if this.m_PacketFuncList.Len() == 0 {
		this.m_PacketFuncList = this.m_pServer.m_PacketFuncList
	}
	//设置连接状态和禁止使用negal算法
	this.m_nState = SSF_CONNECT
	this.m_Conn.(*net.TCPConn).SetNoDelay(true)
	//this.m_Conn.SetKeepAlive(true)
	//this.m_Conn.SetKeepAlivePeriod(5*time.Second)
	//开启一个协程处理信息
	go this.Run()
	if this.m_nConnectType == CLIENT_CONNECT {
		go this.SendLoop()
	}
	return true
}

//发送rpc数据
func (this *ServerSocketClient) Send(head rpc.RpcHead, buff []byte) int {
	defer func() {
		if err := recover(); err != nil {
			base.TraceCode(err)
		}
	}()

	if this.m_nConnectType == CLIENT_CONNECT { //对外链接send不阻塞
		select {
		case this.m_SendChan <- buff:
		default: //网络太卡,tcp send缓存满了并且发送队列也满了
			this.OnNetFail(1)
		}
	} else {
		return this.DoSend(buff)
	}
	return 0
}

//发送数据
func (this *ServerSocketClient) DoSend(buff []byte) int {
	if this.m_Conn == nil {
		return 0
	} else if len(buff) > base.MAX_PACKET {
		panic("send over base.MAX_PACKET")
	}

	n, err := this.m_Conn.Write(buff)
	handleError(err)
	if n > 0 {
		return n
	}

	return 0
}

func (this *ServerSocketClient) OnNetFail(error int) {
	this.Stop()
	if this.m_nConnectType == CLIENT_CONNECT { //netgate对外格式统一
		stream := base.NewBitStream(make([]byte, 32), 32)
		stream.WriteInt(int(DISCONNECTINT), 32)
		stream.WriteInt(int(this.m_ClientId), 32)
		this.HandlePacket(this.m_ClientId, stream.GetBuffer())
	} else {
		this.CallMsg("DISCONNECT", this.m_ClientId)
	}
	if this.m_pServer != nil {
		this.m_pServer.DelClinet(this)
	}
}

func (this *ServerSocketClient) Close() {
	if this.m_nConnectType == CLIENT_CONNECT {
		//close(this.m_SendChan)
	}
	this.Socket.Close()
	if this.m_pServer != nil {
		this.m_pServer.DelClinet(this)
	}
}

func (this *ServerSocketClient) Run() bool {
	var buff = make([]byte, this.m_ReceiveBufferSize)
	loop := func() bool {
		defer func() {
			if err := recover(); err != nil {
				base.TraceCode(err)
			}
		}()

		if this.m_bShuttingDown || this.m_Conn == nil {
			return false
		}

		n, err := this.m_Conn.Read(buff)
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
		if n > 0 {
			//熔断
			if !this.ReceivePacket(this.m_ClientId, buff[:n]) && this.m_nConnectType == CLIENT_CONNECT {
				this.OnNetFail(1)
				return false
			}
		}
		return true
	}

	for {
		if !loop() {
			break
		}
	}

	this.Close()
	fmt.Printf("%s关闭连接", this.m_sIP)
	return true
}

//
func (this *ServerSocketClient) SendLoop() bool {
	for {
		select {
		case buff := <-this.m_SendChan:
			if buff == nil { //信道关闭
				return false
			} else {
				this.DoSend(buff)
			}
		}
	}
	return true
}
