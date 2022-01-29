package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ByronLiang/leader-follower-pattern/examples/campaign/util"

	leaderfollowerpattern "github.com/ByronLiang/leader-follower-pattern"
	"github.com/ByronLiang/leader-follower-pattern/examples/campaign/common"
	"github.com/ByronLiang/leader-follower-pattern/examples/campaign/config"
)

func main() {
	config.InitConfig()
	err := common.InitEtcdClusterClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	lis, err := net.Listen("tcp", config.BindAddress)
	if err != nil {
		log.Printf("listen %s error: %s", config.BindAddress, err.Error())
		return
	}
	mu := http.NewServeMux()
	mu.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("pong"))
	})
	s := &http.Server{Handler: mu}
	go func(server *http.Server) {
		err := server.Serve(lis)
		if err != nil && err != http.ErrServerClosed {
			log.Println("serve error", err)
		}
	}(s)
	defer func() {
		s.Shutdown(context.Background())
		common.CloseEtcdClusterClient()
	}()
	signs := make(chan os.Signal, 1)
	signal.Notify(signs, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL)
	host, err := util.FormatHost(config.RegisterAddress, lis)
	if err != nil {
		log.Printf("format host : %s error: %s\n", config.RegisterAddress, err.Error())
		return
	}
	srv, err := leaderfollowerpattern.NewLeaderFollower(common.EtcdClusterClient, leaderfollowerpattern.Prefix(config.KeyPrefix))
	if err != nil {
		log.Printf("Init LeaderFollower srv error: %s\n", err.Error())
		return
	}
	campaignRes := srv.Campaign(host)
	select {
	case <-signs:
		// 没有成为主服务, 对连接关闭等相关资源回收
		srv.Stop()
		return
	case campaignErr := <-campaignRes:
		if campaignErr != nil {
			// campaign error
			log.Printf("campaign leader srv error: %s\n", campaignErr.Error())
			return
		}
		// call campaignSuccess handle
		log.Printf("host: %s campaign leader success\n", host)
	}
	<-signs
	srv.Stop()
	time.Sleep(1 * time.Second)
	log.Printf("host:%s down\n", host)
}
