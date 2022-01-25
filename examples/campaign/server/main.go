package main

import (
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
	go func() {
		err := http.Serve(lis, mu)
		if err != nil {
			log.Println("serve error", err)
		}
	}()
	defer func() {
		defer lis.Close()
		common.CloseEtcdClusterClient()
	}()
	host, err := util.FormatHost(config.RegisterAddress, lis)
	if err != nil {
		log.Printf("format host : %s error: %s\n", config.RegisterAddress, err.Error())
		return
	}
	cfg := &leaderfollowerpattern.LeaderFollowerConfig{
		HeartBeatTTL: 5,
		Prefix:       config.KeyPrefix,
		Value:        host,
	}
	srv, err := leaderfollowerpattern.NewLeaderFollowerSrv(common.EtcdClusterClient, cfg)
	if err != nil {
		log.Printf("Init LeaderFollower srv error: %s\n", err.Error())
		return
	}
	campaignRes := srv.Campaign()
	campaignErr := <-campaignRes
	if campaignErr != nil {
		// campaign error
		log.Printf("campaign leader srv error: %s\n", campaignErr.Error())
		return
	}
	// call campaignSuccess handle
	log.Printf("host: %s campaign leader success\n", host)
	signs := make(chan os.Signal, 1)
	signal.Notify(signs, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL)
	<-signs
	srv.Stop()
	time.Sleep(1 * time.Second)
	log.Printf("host:%s down\n", host)
}
