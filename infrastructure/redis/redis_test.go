package redis

import (
	"AI_class/def"
	"fmt"
	"testing"
	"time"
)

func TestRedisClient(t *testing.T) {
	client := newRedisClient("test", "127.0.0.1:6379", "")
	err := client.Set("key1", "value1", 60*time.Second)
	if err != nil {
		panic(err)
	}
	err2 := client.Set("key2", "value2", 0)
	if err2 != nil {
		panic(err2)
	}
	val, err := client.Get("key1")
	fmt.Println(val)

}

func TestRedisOpt(t *testing.T) {
	client := newRedisClient("myapp", "localhost:6379", "")
	defer client.ClosePool()

	// ====== String 操作 ======
	client.Set("name", "Alice", 60*time.Second)
	name, _ := client.Get("name")
	fmt.Println(name)
	client.Incr("counter")
	client.SetNxEx("lock:user:123", "locked", def.MilliSecond)

	// ====== Hash 操作 ======
	client.HSet("user:1", "name", "Bob")
	client.HSet("user:1", "age", 25)
	userInfo, _ := client.HGetAll("user:1")
	fmt.Println(userInfo) // map[name:Bob age:25]

	// ====== List 操作 ======
	client.RPush("queue:tasks", "task1", "task2", "task3")
	task, _ := client.LPop("queue:tasks")
	tasks, _ := client.LRange("queue:tasks", 0, -1)
	fmt.Println(task, "\n", tasks)

	// ====== Set 操作 ======
	client.SAdd("tags", "golang", "redis", "database")
	members, _ := client.SMembers("tags")
	isMember, _ := client.SIsMember("tags", "golang") // true
	fmt.Println(members, "\n", isMember)
}
func TestRedisProvider(t *testing.T) {
	client := newRedisClient("myapp", "localhost:6379", "")
	defer client.ClosePool()
	redisP := client.NewPipeline()
	defer redisP.Close()
	redisP.Send("SET", "name", "Alice")
	redisP.Send("GET", "name")
	mesg, err := redisP.Exec()
	msg, err := mesg[0].String()
	fmt.Println(msg, ", err: ", err)
}
