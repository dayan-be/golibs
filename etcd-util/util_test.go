package etcd_util

import (
	"testing"
)

func Test_Get(t *testing.T) {
	ret, err := Get("hello", "world")
	t.Error("ret:", ret, "err:", err)
}