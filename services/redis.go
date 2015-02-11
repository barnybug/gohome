package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisStore struct {
	pool *redis.Pool
}

func NewRedisStore(address string) (Store, error) {
	ret := &RedisStore{newPool(address)}
	// test the connection
	err := ret.Ping()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     1,
		IdleTimeout: 3600 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		// TestOnBorrow: func(c redis.Conn, t time.Time) error {
		// 	_, err := c.Do("PING")
		// 	return err
		// },
	}
}

func (self *RedisStore) Ping() error {
	conn := self.pool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}

func (self *RedisStore) Set(key string, value string) error {
	conn := self.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, value)
	return err
}

func (self *RedisStore) SetWithTTL(key string, value string, ttl uint64) error {
	conn := self.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, value, "EX", ttl)
	return err
}

func (self *RedisStore) Get(key string) (string, error) {
	conn := self.pool.Get()
	defer conn.Close()
	str, err := redis.String(conn.Do("GET", key))
	if err == redis.ErrNil {
		err = errors.New(fmt.Sprint("Key missing: ", key))
	}
	return str, err
}

func (self *RedisStore) GetRecursive(path string) ([]Node, error) {
	conn := self.pool.Get()
	defer conn.Close()
	reply, err := conn.Do("KEYS", path+"/*")
	ikeys := reply.([]interface{})
	keys, err := redis.Strings(reply, err)
	if err != nil {
		return nil, err
	}
	values, err := redis.Strings(conn.Do("MGET", ikeys...))
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, len(keys))
	for i, key := range keys {
		nodes[i] = Node{Key: key, Value: values[i]}
	}
	return nodes, err
}
