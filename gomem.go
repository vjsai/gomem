package main

import (
	"bufio"
        "flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
	"sync"
)

type cacheItem struct{
        data []byte
        accessed time.Time
}
func (item cacheItem) isExpired(expiration time.Duration) bool {
	expiredTime := item.accessed.Add(expiration)
	return time.Now().After(expiredTime)
}
type MyCache struct {
    sync.RWMutex
	data       map[string]cacheItem
	expiration time.Duration
}
func CreateCache(expiration time.Duration) *MyCache {
	return &MyCache{data: make(map[string]cacheItem), expiration: expiration}
}

func (cache *MyCache) Get(key string) (data []byte, ok bool) {
	item, ok := cache.data[key]
	if !ok {
		return nil, false
	}
	if item.isExpired(cache.expiration) {
		cache.Remove(key)
		return nil, false
	}
	item.accessed = time.Now()
	cache.data[key] = item
	return item.data, true
}


func (cache *MyCache) Put(key string, data []byte) {
    cache.Lock()
	cache.data[key] = cacheItem{data, time.Now()}
	cache.Unlock()
}


func (cache *MyCache) Add(key string,data []byte) (n_data []byte,ok bool) {
     cache.Lock()
     _,ok = cache.Get(key)
     if !ok {
         cache.Unlock()
         return nil,false
     }
     cache.Put(key,data)
     cache.Unlock()
     return  nil,true
}

func (cache *MyCache) Remove(key string) {
	delete(cache.data, key)
}

func (cache *MyCache) Clear() {
	cache.data = make(map[string]cacheItem)
}

func (cache *MyCache) RemoveExpired() {
	for key, item := range cache.data {
		if item.isExpired(cache.expiration) {
			cache.Remove(key)
		}
	}
}


func (cache *MyCache) Increment(k string, n int64) {
	//need to implement INCR
}

func (cache *MyCache) Touch(key string) {
	//need to implement update of expiration time without fetching
}

func (cache *MyCache) Decrement(k string, n int64) {
	//need to implement
}

func (cache *MyCache) Flush_All()
{
   //flush the invalid data
}

/*
 need to implement functions for stats
 provide different caching times.
 */

var (
  singleFlag = flag.Bool("single", false, "Start in single mode")
  m_cache =    CreateCache(time.Hour)
  port = flag.String("port", "11211", "port number")
)
func main() {
    flag.Parse()

	listener, err := net.Listen("tcp", "127.0.0.1:"+ *port)
	if err != nil {
		panic("Error listening on "+ *port + ":"+ err.Error())
	}

	if *singleFlag {
		netconn, err := listener.Accept()
		if err != nil {
			panic("Accept error: " + err.Error())
		}

		handleConn(netconn)

	} else {
		for {
			netconn, err := listener.Accept()
			if err != nil {
				panic("Accept error: " + err.Error())
			}

			go handleConn(netconn)
		}
	}

}

/*
 * Networking
 */
func handleConn(conn net.Conn) {
    defer conn.Close()
	reader := bufio.NewReader(conn)
	for {

		// Fetch

		content, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err)
			return
		}

		content = content[:len(content)-2] // Chop \r\n

		// Handle

		parts := strings.Split(content, " ")
		cmd := parts[0]
		switch cmd {

		case "get":
			key := parts[1]
			g_value,g_ok := m_cache.Get(key)
			g_length := strconv.Itoa(len(g_value))
			if g_ok{
				    conn.Write([]uint8("VALUE " + string(g_value) + " 0 " + g_length  + "\r\n"))
				    conn.Write([]uint8(string(g_value)  + "\r\n"))
				    conn.Write([]uint8("END\r\n"))
			} else {
			        conn.Write([]uint8("ERROR\r\n"))
			}


		case "set":
			key := parts[1]
			//exp := parts[2]
			//flags := parts[3]
			length, _ := strconv.Atoi(parts[4])
			// Really we should read exactly 'length' bytes + \r\n
			val := make([]byte, length)
			reader.Read(val)
			m_cache.Put(key,val)
			conn.Write([]uint8("STORED\r\n"))


		case "version":
		      conn.Write([]uint8("VERSION 1.0\r\n"))

		case "delete":
		      key := parts[1]
		      m_cache.Remove(key)
		      conn.Write([]uint8("DELETED\r\n"))


		case "quit":
		      conn.Write([]uint8("Closing connection with gomem\r\n"))
		      conn.Close()
		}
	}
}

