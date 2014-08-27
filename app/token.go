package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/EPICPaaS/go-uuid/uuid"
	"github.com/EPICPaaS/gopush-cluster/ketama"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"strconv"
	"strings"
	"time"
)

var RedisNoConnErr = errors.New("can't get a redis conn")

type RedisDelToken struct {
	expires map[string]int64
}

type redisStorage struct {
	pool  map[string]*redis.Pool
	ring  *ketama.HashRing
	delCH chan *RedisDelToken
}

var rs *redisStorage

type RedisToken struct {
	Token  string
	Expire int64
}

// initRedisStorage initialize the redis pool and consistency hash ring.
func InitRedisStorage() {
	glog.Info("Connecting Redis....")

	var (
		err error
		w   int
		nw  []string
	)

	redisPool := map[string]*redis.Pool{}
	ring := ketama.NewRing(Conf.RedisKetamaBase)

	for n, addr := range Conf.RedisSource {
		nw = strings.Split(n, ":")
		if len(nw) != 2 {
			err = errors.New("node config error, it's nodeN:W")
			glog.Errorf("strings.Split(\"%s\", :) failed (%v)", n, err)
			panic(err)
		}

		w, err = strconv.Atoi(nw[1])
		if err != nil {
			glog.Errorf("strconv.Atoi(\"%s\") failed (%v)", nw[1], err)
			panic(err)
		}

		tmp := addr

		redisPool[nw[0]] = &redis.Pool{
			MaxIdle:     Conf.RedisMaxIdle,
			MaxActive:   Conf.RedisMaxActive,
			IdleTimeout: Conf.RedisIdleTimeout,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial("tcp", tmp)
				if err != nil {
					glog.Errorf("redis.Dial(\"tcp\", \"%s\") error(%v)", tmp, err)
					return nil, err
				}

				return conn, err
			},
		}

		ring.AddNode(nw[0], w)
	}

	ring.Bake()
	rs = &redisStorage{pool: redisPool, ring: ring, delCH: make(chan *RedisDelToken, 10240)}
	go rs.clean()

	glog.Info("Redis connected")

}

// 根据令牌返回用户.
func getUserByToken(token string) *member {
	conn := rs.getConn("token")
	if conn == nil {
		return nil
	}
	defer conn.Close()

	idx := strings.Index(token, "_")
	if -1 == idx {
		return nil
	}

	uid := token[:idx]
	// TODO: validate token

	expireTime := int64(Conf.TokenExpire) + time.Now().Unix()

	values, err := redis.Values(conn.Do("ZRANGEBYSCORE", "token", "-inf", fmt.Sprintf("%d", expireTime), "WITHSCORES"))
	if err != nil {
		glog.Error(err)

		return nil
	}

	expires := map[string]int64{}

	for len(values) > 0 {
		expire := int64(0)
		b := []byte{}
		values, err = redis.Scan(values, &b, &expire)
		if err != nil {
			glog.Errorf("redis.Scan() error(%v)", err)
			return nil
		}

		rt := &RedisToken{}
		if err := json.Unmarshal(b, rt); err != nil {
			glog.Errorf("json.Unmarshal(\"%s\", rt) error(%v)", string(b), err)
			expires[rt.Token] = expire // 转 JSON 异常的也认为过期

			continue
		}

		if rt.Expire < expireTime { // 如果该令牌已经过期
			expires[rt.Token] = expire
		}
	}

	if len(expires) > 0 {
		select {
		case rs.delCH <- &RedisDelToken{expires}:
		default:
			glog.Warning("token channel is full")
		}
	}

	return getUserByUid(uid)
}

// 令牌生成.
func genToken(user *member) (string, error) {
	conn := rs.getConn(user.Uid)

	if conn == nil {
		return "", RedisNoConnErr
	}

	defer conn.Close()

	expire := int64(Conf.TokenExpire) + time.Now().Unix()
	token := user.Uid + "_" + uuid.New()

	rt := &RedisToken{token, expire}
	m, err := json.Marshal(rt)

	if err != nil {
		glog.Errorf("json.Marshal(\"%v\") error(%v)", rt, err)
		return "", err
	}

	if err := conn.Send("ZADD", "token", expire, m); err != nil {
		glog.Errorf("conn.Send(\"ZADD\", \"%s\", %d, \"%s\") error(%v)", "token", expire, string(m), err)
		return "", err
	}

	if err := conn.Flush(); err != nil {
		glog.Errorf("conn.Flush() error(%v)", err)
		return "", err
	}

	_, err = conn.Receive()
	if err != nil {
		glog.Errorf("conn.Receive() error(%v)", err)
		return "", err
	}

	return token, nil
}

// 获取 Redis 连接.
func (s *redisStorage) getConn(key string) redis.Conn {
	if len(s.pool) == 0 {
		return nil
	}

	node := s.ring.Hash(key)
	p, ok := s.pool[node]
	if !ok {
		glog.Warningf("key: \"%s\" hit redis node: \"%s\" not in pool", key, node)
		return nil
	}

	glog.V(5).Infof("key: \"%s\" hit redis node: \"%s\"", key, node)

	return p.Get()
}

// 清理过期令牌.
func (s *redisStorage) clean() {
	for {
		info := <-s.delCH
		conn := s.getConn("token")

		if conn == nil {
			glog.Warning("get redis connection nil")
			continue
		}

		for token, expire := range info.expires {
			if err := conn.Send("ZREMRANGEBYSCORE", "token", expire, expire); err != nil {
				glog.Errorf("conn.Send(\"ZREMRANGEBYSCORE\", \"%s\", %d, %d) error(%v)", "token", expire, expire, err)
				conn.Close()
				continue
			}

			glog.V(3).Infof("Token [%s] expired", token)
		}

		if err := conn.Flush(); err != nil {
			glog.Errorf("conn.Flush() error(%v)", err)
			conn.Close()
			continue
		}

		for _, _ = range info.expires {
			_, err := conn.Receive()
			if err != nil {
				glog.Errorf("conn.Receive() error(%v)", err)
				conn.Close()
				continue
			}
		}

		conn.Close()
	}
}
