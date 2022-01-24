package leader_follower_pattern

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Watcher struct {
	ctx      context.Context
	cancel   context.CancelFunc
	session  *concurrency.Session
	election *concurrency.Election
	prefix   string
}

func NewWatcher(cctx context.Context, pfx string, c *clientv3.Client) (*Watcher, error) {
	ctx, cancel := context.WithCancel(cctx)
	w := &Watcher{
		ctx:    ctx,
		cancel: cancel,
		prefix: pfx,
	}
	prefix := w.prefix + "/"
	resp, err := c.Get(ctx, prefix, clientv3.WithFirstCreate()...)
	if err != nil {
		return nil, err
	} else if len(resp.Kvs) == 0 {
		// no leader currently elected
		return nil, concurrency.ErrElectionNoLeader
	}
	session, err := concurrency.NewSession(c)
	if err != nil {
		return nil, err
	}
	w.session = session
	w.election = concurrency.ResumeElection(w.session, prefix,
		string(resp.Kvs[0].Key), resp.Kvs[0].CreateRevision)
	return w, nil
}

func (w *Watcher) Watch(watchHandle func(res clientv3.GetResponse)) (string, error) {
	currentRes, err := w.election.Leader(w.ctx)
	if err != nil {
		return "", err
	}
	go func() {
		// 原生SDK选举监听
		resChan := w.election.Observe(w.ctx)
		for {
			select {
			case <-w.ctx.Done():
				return
			case res, ok := <-resChan:
				if !ok {
					return
				}
				watchHandle(res)
			}
		}
	}()
	return string(currentRes.Kvs[0].Value), nil
}

func (w *Watcher) WatchWithEventType(watchHandle func(res ObserveRes)) (string, error) {
	currentRes, err := w.election.Leader(w.ctx)
	if err != nil {
		return "", err
	}
	resChan := w.Observe(w.ctx)
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case res, ok := <-resChan:
				if !ok {
					return
				}
				watchHandle(res)
			}
		}
	}()
	return string(currentRes.Kvs[0].Value), nil
}

func (w *Watcher) Stop() error {
	w.cancel()
	return w.session.Close()
}
