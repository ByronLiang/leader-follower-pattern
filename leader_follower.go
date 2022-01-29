package leader_follower_pattern

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type leaderFollower struct {
	ctx      context.Context
	cancel   context.CancelFunc
	session  *concurrency.Session
	election *concurrency.Election
	opts     *options
}

type options struct {
	heartBeatTTL int
	prefix       string
}

type option func(o *options)

func HeartBeatTTL(ttl int) option {
	return func(o *options) {
		o.heartBeatTTL = ttl
	}
}

func Prefix(prefix string) option {
	return func(o *options) {
		o.prefix = prefix
	}
}

func NewLeaderFollower(client *clientv3.Client, op ...option) (*leaderFollower, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := &options{heartBeatTTL: 5}
	for _, op := range op {
		op(opts)
	}
	// 若正常退出, 触发resign  而 ttl 针对非resign 下故障容忍时长触发重新选举
	sess, err := concurrency.NewSession(client, concurrency.WithTTL(opts.heartBeatTTL))
	if err != nil {
		return nil, err
	}
	ele := concurrency.NewElection(sess, opts.prefix)
	return &leaderFollower{
		ctx:      ctx,
		session:  sess,
		election: ele,
		cancel:   cancel,
		opts:     opts,
	}, nil
}

func (srv *leaderFollower) GetValue() ([]byte, error) {
	res, err := srv.election.Leader(srv.ctx)
	if err != nil {
		return nil, err
	}
	return res.Kvs[0].Value, nil
}

func (srv *leaderFollower) Campaign(val string) chan error {
	campaignRes := make(chan error)
	go func() {
		err := srv.election.Campaign(srv.ctx, val)
		campaignRes <- err
		close(campaignRes)
	}()
	return campaignRes
}

func (srv *leaderFollower) Stop() error {
	srv.election.Resign(srv.ctx)
	srv.cancel()
	return srv.session.Close()
}
