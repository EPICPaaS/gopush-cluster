package app

import (
	"encoding/json"
	"errors"
	"github.com/EPICPaaS/go-uuid/uuid"
	"github.com/EPICPaaS/gopush-cluster/ketama"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"strconv"
	"strings"
	"time"
)

var RedisNoConnErr = errors.New("can't get a redis conn")

// Struct for delele token.
type RedisDelToken struct {
	expires []int64
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
		// WARN: closures use
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

	glog.Info("Redis connected")

	// TODO: token clean: go redis.clean()
}

// 根据令牌返回用户.
func getUserByToken(token string) *member {
	uid := token[:strings.Index(token, "_")]
	// TODO: validate token

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
