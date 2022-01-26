package leader_follower_pattern

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type leaderFollowerSrv struct {
	ctx      context.Context
	cancel   context.CancelFunc
	session  *concurrency.Session
	election *concurrency.Election
	prefix   string
	value    string
}

type LeaderFollowerConfig struct {
	HeartBeatTTL int
	Prefix       string
	Value        string
}

func NewLeaderFollowerSrv(client *clientv3.Client, config *LeaderFollowerConfig) (*leaderFollowerSrv, error) {
	ctx, cancel := context.WithCancel(context.Background())
	// 若正常退出, 触发resign  而 ttl 针对非resign 下故障容忍时长触发重新选举
	sess, err := concurrency.NewSession(client, concurrency.WithTTL(config.HeartBeatTTL))
	if err != nil {
		return nil, err
	}
	ele := concurrency.NewElection(sess, config.Prefix)
	return &leaderFollowerSrv{
		ctx:      ctx,
		session:  sess,
		election: ele,
		cancel:   cancel,
		prefix:   config.Prefix,
		value:    config.Value,
	}, nil
}

func (srv *leaderFollowerSrv) GetValue() string {
	return srv.value
}

func (srv *leaderFollowerSrv) Campaign() chan error {
	campaignRes := make(chan error)
	go func() {
		err := srv.election.Campaign(srv.ctx, srv.value)
		campaignRes <- err
	}()
	return campaignRes
}

func (srv *leaderFollowerSrv) Stop() error {
	srv.election.Resign(srv.ctx)
	srv.cancel()
	return srv.session.Close()
}
