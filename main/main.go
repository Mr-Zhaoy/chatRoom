package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// 创建用户结构体
type Client struct {
	C    chan string
	Name string
	Add  string
}

// 创建全局map，存储在线用户， 初始化map
var Onlinemap = make(map[string]Client)

// 创建全局mchannel 传递用户消息
var message = make(chan string)

// 管理在线用户,把全局message内容写给用户
func Manage() {
	// 监听全局message,吧message数据写到msg
	for {
		msg := <-message
		for _, client := range Onlinemap {
			client.C <- msg
		}
	}
}

// 读取数据到终端
func WriteToClient(conn net.Conn, client Client) {
	// 监听 用户自带Channel 上是否有消息。
	for msg := range client.C {
		//!!!\n
		conn.Write([]byte(msg + "\n"))
	}
}

// 数据信息
func Makemsg(client Client, msg string) (msg1 string) {
	msg1 = "[" + client.Add + "]" + client.Name + ":" + msg
	return
}

func HandlerConnect(conn net.Conn) {
	defer conn.Close()
	//创建一个channel 判断用户是否活跃
	hasDate := make(chan bool)
	// 获取用户 网络地址 IP+port
	addr := conn.RemoteAddr().String()
	// 创建新连接用户的 结构体. 默认用户是 IP+port
	client := Client{make(chan string), addr, addr}
	// 将新连接用户，添加到在线用户map中. key: IP+port value：client
	Onlinemap[addr] = client
	// 创建专门用来给当前 用户发送消息的 go 程
	go WriteToClient(conn, client)
	// 发送 用户上线消息到 全局channel 中
	message <- Makemsg(client, "Login")
	// 创建一个 channel ， 用来判断用退出状态
	isQuit := make(chan bool)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				isQuit <- true
				fmt.Printf("检测到客户端%s退出\n", client.Name)
				return
			}
			if err != nil {
				fmt.Println("conn.Read err", err)
				return
			}
			msg := string(buf[:n-1])
			//提取在线用户
			if "who\n" == msg {
				conn.Write([]byte("online user list:\n"))
				// 遍历当前 map ，获取在线用户
				for _, user := range Onlinemap {
					userInfo := user.Add + user.Name
					conn.Write([]byte(userInfo + "\n"))
				}
				// 判断用户发送了 改名 命令
			} else if string(buf[:n-1]) == "rename" && len(string(buf[:n-1])) > 8 { //rename
				newName := strings.Split(msg, "|")
				client.Name = newName[1]
				Onlinemap[addr] = client
				conn.Write([]byte("rename successful\n"))
			} else {
				// 将读到的用户消息，写入到message中。
				message <- Makemsg(client, msg)
			}
			hasDate <- true
		}

	}()
	// 保证 不退出
	for {
		// 监听channel上的数据流动
		select {
		case <-isQuit:
			delete(Onlinemap, client.Add)        // 将用户从 online移除
			message <- Makemsg(client, "logout") // 写入用户退出消息到全局channel
			return
		case <-hasDate:
			// 什么都不做。 目的是重置 下面 case 的计时器。
		case <-time.After(time.Second * 10):
			delete(Onlinemap, client.Add)        // 将用户从 online移除
			message <- Makemsg(client, "logout") // 写入用户退出消息到全局channel
			return
		}
	}
}

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:8006")
	if err != nil {
		fmt.Println("net.Listen err", err)
		return
	}
	defer listen.Close()
	//创建管理者go程
	go Manage()

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Accept() err", err)
			return
		}
		go HandlerConnect(conn)
	}

}
