package leader_follower_pattern

import (
	"context"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ObserveRes struct {
	Res       clientv3.GetResponse
	EventType mvccpb.Event_EventType
}

// 自定义封装监听重新选举与无法获取选取等事件
func (w *Watcher) Observe(ctx context.Context) chan ObserveRes {
	ch := make(chan ObserveRes)
	go w.observe(ctx, ch)
	return ch
}

func (w *Watcher) observe(ctx context.Context, ch chan ObserveRes) {
	client := w.session.Client()
	keyPrefix := w.prefix + "/"
	defer close(ch)
	for {
		resp, err := client.Get(ctx, keyPrefix, clientv3.WithFirstCreate()...)
		if err != nil {
			return
		}

		var kv *mvccpb.KeyValue
		var hdr *pb.ResponseHeader

		if len(resp.Kvs) == 0 {
			cctx, cancel := context.WithCancel(ctx)
			// wait for first prefix put on prefix
			opts := []clientv3.OpOption{clientv3.WithRev(resp.Header.Revision), clientv3.WithPrefix()}
			wch := client.Watch(cctx, keyPrefix, opts...)
			for kv == nil {
				wr, ok := <-wch
				if !ok || wr.Err() != nil {
					cancel()
					return
				}
				// only accept puts; a delete will make observe() spin
				for _, ev := range wr.Events {
					if ev.Type == mvccpb.PUT {
						hdr, kv = &wr.Header, ev.Kv
						// may have multiple revs; hdr.rev = the last rev
						// set to kv's rev in case batch has multiple Puts
						hdr.Revision = kv.ModRevision
						break
					}
				}
			}
			cancel()
		} else {
			hdr, kv = resp.Header, resp.Kvs[0]
		}
		res := ObserveRes{
			Res:       clientv3.GetResponse{Header: hdr, Kvs: []*mvccpb.KeyValue{kv}},
			EventType: mvccpb.PUT,
		}
		select {
		case ch <- res:
		case <-ctx.Done():
			return
		}

		cctx, cancel := context.WithCancel(ctx)
		wch := client.Watch(cctx, string(kv.Key), clientv3.WithRev(hdr.Revision+1))
		keyDeleted := false
		for !keyDeleted {
			wr, ok := <-wch
			if !ok {
				cancel()
				return
			}
			for _, ev := range wr.Events {
				resp.Header = &wr.Header
				resp.Kvs = []*mvccpb.KeyValue{ev.Kv}
				res := ObserveRes{
					Res:       clientv3.GetResponse{Header: resp.Header, Kvs: resp.Kvs},
					EventType: ev.Type,
				}
				select {
				case ch <- res:
				case <-cctx.Done():
					cancel()
					return
				}
				if ev.Type == mvccpb.DELETE {
					keyDeleted = true
					break
				}
			}
		}
		cancel()
	}
}
