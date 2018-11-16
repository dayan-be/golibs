package etcd_util

import (
	"fmt"
	"time"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
)

type Client struct {
	clientv3.KV  // etcd KV
}

var DialTimeout = 5 * time.Second

// defaultEtcdEndpoints used for local testing as /etc/ss_conf/etcd.conf not always present
var defaultEtcdEndpoints []string = []string{"10.4.17.150:4001", "10.4.17.151:4001", "10.4.17.152:4001"}

// NewClient create client from ss_conf
func NewClient() (c *Client, err error) {
	// Parse endpoints
	hostPorts, err := ssconf.GetServerList("/etc/ss_conf/etcd.conf", "etcd_host_port")
	if err != nil {
		logrus.Error("Error get etcd endpoints from ss_conf", err)
		hostPorts = defaultEtcdEndpoints
	}
	if len(hostPorts) < 1 {
		return nil, fmt.Errorf("/etc/ss_conf/etcd.conf hostPorts empty")
	}
	endpoints := make([]string, len(hostPorts))
	for i, e := range hostPorts {
		endpoints[i] = fmt.Sprintf("http://%s", e)
	}

	logrus.Info("etcd endpoints, %v", endpoints)
	return NewClientWithEndpoints(endpoints)
}

// NewClientWithEndpoints create client from endpoints
func NewClientWithEndpoints(endpoints []string) (c *Client, err error) {
	// Create etcd client
	client, err := clientv3.New(clientv3.Config{
		Endpoints:               endpoints,
		DialTimeout: DialTimeout,
	})
	if err != nil {
		logrus.Error("Can't create etcd client", err)
		return nil, err
	}

	// Create Client
	c = &Client{
		clientv3.NewKV(client),
	}
	return c, nil
}
