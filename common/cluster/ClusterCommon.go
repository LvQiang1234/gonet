package cluster

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"gonet/actor"
	"gonet/base"
	"gonet/common"
	"gonet/common/cluster/etv3"
	"gonet/rpc"
	"strings"
)

const(
	ETCD_DIR =  "server/"
)

type(
	Service etv3.Service
	Master etv3.Master
	Snowflake etv3.Snowflake
)

//注册服务器
func NewService(info *common.ClusterInfo, Endpoints []string) *Service{
	service := &etv3.Service{}
	service.Init(info, Endpoints)
	return (*Service)(service)
}

//监控服务器
func NewMaster(info common.IClusterInfo, Endpoints []string, pActor actor.IActor) *Master {
	master := &etv3.Master{}
	master.Init(info, Endpoints, pActor)
	return (*Master)(master)
}

//uuid生成器
func NewSnowflake(Endpoints []string) *Snowflake{
	uuid := &etv3.Snowflake{}
	uuid.Init(Endpoints)
	return (*Snowflake)(uuid)
}

func getChannel(clusterInfo common.ClusterInfo) string {
	return fmt.Sprintf("%s/%s/%d", ETCD_DIR, clusterInfo.String(), clusterInfo.Id())
}

func getTopicChannel(clusterInfo common.ClusterInfo) string {
	return fmt.Sprintf("%s/%s", ETCD_DIR, clusterInfo.String())
}

func getRpcChannel(head rpc.RpcHead) string {
	return fmt.Sprintf("%s/%s/%d", ETCD_DIR, strings.ToLower(head.DestServerType.String()), head.ClusterId)
}

func getRpcTopicChannel(head rpc.RpcHead) string {
	return fmt.Sprintf("%s/%s", ETCD_DIR, strings.ToLower(head.DestServerType.String()))
}

func setupNatsConn(connectString string, appDieChan chan bool, options ...nats.Option) (*nats.Conn, error) {
	natsOptions := append(
		options,
		nats.DisconnectHandler(func(_ *nats.Conn) {
			base.GLOG.Println("disconnected from nats!")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			base.GLOG.Printf("reconnected to nats server %s with address %s in cluster %s!", nc.ConnectedServerId(), nc.ConnectedAddr(), nc.ConnectedUrl() )
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			err := nc.LastError()
			if err == nil {
				base.GLOG.Println("nats connection closed with no error.")
				return
			}

			base.GLOG.Fatalf("nats connection closed. reason: %q", nc.LastError())
			if appDieChan != nil {
				appDieChan <- true
			}
		}),
	)

	nc, err := nats.Connect(connectString, natsOptions...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

