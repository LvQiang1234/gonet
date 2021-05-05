package main

import (
	"fmt"
	"gonet/actor"
	"gonet/base"
	"gonet/common"
	"gonet/server/account"
	"gonet/server/netgate"
	"gonet/server/world"
	"gonet/server/world/chat"
	"gonet/server/world/cmd"
	"gonet/server/world/data"
	"gonet/server/world/mail"
	"gonet/server/world/player"
	"gonet/server/world/social"
	"gonet/server/world/toprank"
	"os"
	"os/signal"
	"syscall"
)

func InitMgr(serverName string) {
	//一些共有数据量初始化
	common.Init()
	if serverName == "account" {
	} else if serverName == "netgate" {
	} else if serverName == "world" {
		cmd.Init()
		data.InitRepository()
		player.MGR.Init(1000)
		chat.MGR.Init(1000)
		mail.MGR.Init(1000)
		toprank.MGR().Init(1000)
		player.SIMPLEMGR.Init(1000)
		social.MGR().Init(1000)
		actor.MGR.InitActorHandle(world.SERVER.GetCluster())
	}
}

//程序退出后执行
func ExitMgr(serverName string) {
	if serverName == "account" {
	} else if serverName == "netgate" {
	} else if serverName == "world" {
	}
}

func main() {
	args := os.Args
	if args[0] == "account" {
		account.SERVER.Init()
	} else if args[0] == "netgate" {
		netgate.SERVER.Init()
	} else if args[0] == "world" {
		world.SERVER.Init()
	}

	base.SEVERNAME = args[0]

	InitMgr(args[0])

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-c

	ExitMgr(args[0])
	fmt.Printf("server【%s】 exit ------- signal:[%v]", args[0], s)
}
