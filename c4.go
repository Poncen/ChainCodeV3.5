//客户端程序，用于向某一节点发送消息请求，采用了tcp长链接
//消息类型：
//1.join，加入操作
//2.leave，离开操作
//3.quit，关闭客户端
//4.message，上链消息

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Info struct {
	Id   string
	Port string
}

type message struct {
	Kind      string
	Verifykey bool
	Jlmessage *Info
	Peerinfo  map[string]Info
	Talk      string
	Hashtable []string
}

//定义通道
var ch chan int = make(chan int)

//定义ID
var id, port, hash string

func reader(conn *net.TCPConn) {
	buff := make([]byte, 128)
	for {
		j, err := conn.Read(buff)
		if err != nil {
			ch <- 1
			break
		}

		fmt.Println(string(buff[0:j]))
	}
}

func input() (id string, port string) {
	fmt.Println("Input your listenport")
	fmt.Scanln(&port)
	fmt.Println("Input your id")
	fmt.Scanln(&id)
	fmt.Println("目标port为:", port, "目标id为", id)
	return id, port
}

//main
func main() {
	var key bool
	key = true
	fmt.Println("Input your server's listen port")
	fmt.Scanln(&port)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:"+port)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	b := &message{
		Kind: "msg",
		Talk: "client connected",
	}
	jsonmake(b, conn)
	if err != nil {
		fmt.Println("Server is not starting")
		os.Exit(0)
	}
	defer conn.Close()
	go reader(conn)
	defer func() {
		b := &message{
			Kind: "msg",
			Talk: "client disconnected",
		}
		jsonmake(b, conn)
		conn.Close()
	}()

	for {
		var msg string
		fmt.Scanln(&msg)
		if msg == "link" {
			b := &message{
				Kind: "link",
			}
			jsonmake(b, conn)
		}
		if msg == "join" { //以后会提取出独立的函数
			id, port = input()
			a := &Info{
				id,
				port,
			}
			b := &message{
				Kind:      "join",
				Verifykey: key,
				Jlmessage: a,
			}
			// c, err := json.Marshal(b)
			// if err != nil {
			// 	fmt.Println("err:", err)
			// }
			// conn.Write(c)
			jsonmake(b, conn)
		}
		if msg == "leave" {
			id, port = input()
			a := &Info{
				id,
				port,
			}
			b := &message{
				Kind:      "leave",
				Verifykey: key,
				Jlmessage: a,
			}
			jsonmake(b, conn)
		}
		if msg == "quit" {
			break
		}
		if msg == "work" {
			fmt.Scanln(&hash)
			b := &message{
				Kind:      "work",
				Verifykey: key,
				Talk:      hash,
			}
			jsonmake(b, conn)
		}

		//select 为非阻塞的
		select {
		case <-ch:
			fmt.Println("Server错误!请重新连接!")
			os.Exit(1)
		default:
		}
	}
}

func jsonmake(b *message, conn *net.TCPConn) {
	c, err := json.Marshal(b)
	if err != nil {
		fmt.Println("err:", err)
	}
	conn.Write(c)
}
