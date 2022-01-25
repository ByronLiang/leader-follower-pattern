package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/ByronLiang/leader-follower-pattern/examples/campaign/config"

	"github.com/ByronLiang/leader-follower-pattern/examples/campaign/common"

	leaderfollowerpattern "github.com/ByronLiang/leader-follower-pattern"
)

func main() {
	config.InitConfig()
	err := common.InitEtcdClusterClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	NewLeaderFollowerSrvWatcher()
	common.CloseEtcdClusterClient()
}

func NewLeaderFollowerSrvWatcher() {
	ctx := context.Background()
	watcher, err := leaderfollowerpattern.NewWatcher(ctx, config.KeyPrefix, common.EtcdClusterClient)
	if err != nil {
		log.Printf("lead srv watch error: %s\n", err.Error())
		return
	}
	var currentVal string
	watchVal, err := watcher.Watch(func(res clientv3.GetResponse) {
		currentVal = string(res.Kvs[0].Value)
	})
	if err != nil {
		return
	}
	currentVal = watchVal
	signs := make(chan os.Signal, 1)
	signal.Notify(signs, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL)
	ticker := time.NewTicker(10 * time.Second)
blockFor:
	for {
		select {
		case <-ticker.C:
			url := fmt.Sprintf("http://%s/ping", currentVal)
			res, err := http.Get(url)
			if err != nil {
				log.Printf("http get %s error: %s\n", url, err)
				continue
			}
			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Printf("read body error: %s\n", err)
				continue
			} else {
				log.Printf("get from %s data: %s", currentVal, string(data))
			}
			res.Body.Close()
			log.Printf("watch host: %s \n", currentVal)
		case <-signs:
			log.Println("end watcher")
			break blockFor
		}
	}
	watcher.Stop()
	time.Sleep(1 * time.Second)
}
