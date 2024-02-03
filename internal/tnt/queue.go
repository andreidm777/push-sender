package tnt

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	tarantool "github.com/tarantool/go-tarantool/v2"
	"github.com/tarantool/go-tarantool/v2/pool"
	"github.com/tarantool/go-tarantool/v2/queue"

	log "github.com/sirupsen/logrus"
)

type QueueConnectionHandler struct {
	name string
	cfg  queue.Cfg

	err       error
	mutex     sync.Mutex
	updated   chan struct{}
	masterCnt int32
}

// QueueConnectionHandler implements the ConnectionHandler interface.
var _ pool.ConnectionHandler = &QueueConnectionHandler{}

// NewQueueConnectionHandler creates a QueueConnectionHandler object.
func NewQueueConnectionHandler(name string, cfg queue.Cfg) *QueueConnectionHandler {
	return &QueueConnectionHandler{
		name:    name,
		cfg:     cfg,
		updated: make(chan struct{}, 10),
	}
}

// Discovered configures a queue for an instance and identifies a shared queue
// session on master instances.
//
// NOTE: the Queue supports only a master-replica cluster configuration. It
// does not support a master-master configuration.
func (h *QueueConnectionHandler) Discovered(id string, conn *tarantool.Connection,
	role pool.Role) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.err != nil {
		return h.err
	}

	master := role == pool.MasterRole

	_ = queue.New(conn, h.name)

	defer func() {
		h.updated <- struct{}{}
	}()

	// The queue only works with a master instance.
	if !master {
		return nil
	}

	log.Debugf("Master %s is ready to work!", id)
	atomic.AddInt32(&h.masterCnt, 1)

	return nil
}

// Deactivated doesn't do anything useful for the example.
func (h *QueueConnectionHandler) Deactivated(id string, conn *tarantool.Connection,
	role pool.Role) error {
	if role == pool.MasterRole {
		atomic.AddInt32(&h.masterCnt, -1)
	}
	return nil
}

// Closes closes a QueueConnectionHandler object.
func (h *QueueConnectionHandler) Close() {
	close(h.updated)
}

type TntCfg struct {
	Addrs    []string
	User     string
	Password string
	Timeout  time.Duration
	Ttl      time.Duration
}

type Queue interface {
	TakeTimeout(timeout time.Duration) (*queue.Task, error)
}

type TntQueue struct {
	name    string
	handler *QueueConnectionHandler
	queue   queue.Queue
}

func NewTntQueue(ctx context.Context, name string, cfg *TntCfg) (tqueue Queue, err error) {
	tntQueue := &TntQueue{
		name: name,
	}

	tqueue = tntQueue

	qCfg := queue.Cfg{
		Temporary:   false,
		IfNotExists: false,
		Kind:        queue.FIFO_TTL,
		Opts: queue.Opts{
			Ttl: cfg.Ttl,
		},
	}

	tntQueue.handler = NewQueueConnectionHandler(name, qCfg)

	poolInstances := []pool.Instance{}
	connOpts := tarantool.Opts{
		Timeout: cfg.Timeout,
	}
	for _, serv := range cfg.Addrs {
		dialer := tarantool.NetDialer{
			Address:  serv,
			User:     cfg.User,
			Password: cfg.Password,
		}
		poolInstances = append(poolInstances, pool.Instance{
			Name:   serv,
			Dialer: dialer,
			Opts:   connOpts,
		})

	}

	poolOpts := pool.Opts{
		CheckTimeout:      5 * time.Second,
		ConnectionHandler: tntQueue.handler,
	}

	connPool, err := pool.ConnectWithOpts(ctx, poolInstances, poolOpts)

	if err != nil {
		log.Errorf("Unable to connect to the pool: %s", err)
		return
	}

	// Wait for a queue initialization and master instance identification in
	// the queue.
	for range cfg.Addrs {
		<-tntQueue.handler.updated
	}

	if tntQueue.handler.err != nil {
		log.Errorf("Unable to identify in the pool: %s", tntQueue.handler.err)
		err = tntQueue.handler.err
		return
	}

	// Create a Queue object from the ConnectionPool object via
	// a ConnectorAdapter.
	rw := pool.NewConnectorAdapter(connPool, pool.RW)
	tntQueue.queue = queue.New(rw, name)

	return
}

func (tntQueue *TntQueue) TakeTimeout(timeout time.Duration) (*queue.Task, error) {
	return tntQueue.queue.TakeTimeout(timeout)
}
