//服务器端代码，处理来自客户端的请求，本地包含了节点拓扑表InfoMap，连接池ConnMap，信息存储表HashTable
//采用Tcp长连接，和全部的节点以及中心节点都维护有一个连接通道
//制定了统一的结构体massage，用于各种需求下的消息传递
//采用json做为消息的序列化方法
//另：
//1.心跳检测：定期向中心节点端发送确认消息，证明连接的完整性
//2.超时处理：如果中心节点在一定时间内没返回确认消息，返回一个错误信息

//目标：反馈连接，采用通道阻塞方式，每次消息都返回一个确认信息

package main

import (
	"encoding/json"
	"fmt"
	"net"
)

//定义节点信息结构体info
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

type Link struct {
	Remoteaddr string
	Localaddr  string
	Conninfo   *net.TCPConn
}

//全局变量定义
var response, key bool      //中心节点是否通过
var port, id string         //本地ip
var InfoMap map[string]Info //本地拓扑
var myslice []string
var ConnMap map[string]Link //连接池
var tcpConn *net.TCPConn
var ch chan bool = make(chan bool) //通道阻塞，确保消息顺序的正确

//错误检查函数
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

//json传输
func jsonmake(b *message, conn *net.TCPConn) {
	c, err := json.Marshal(b)
	if err != nil {
		fmt.Println("err:", err)
	}
	conn.Write(c)
}

//输入信息
func input() (id string, port string) {
	fmt.Println("Input your listenport")
	fmt.Scanln(&port)
	fmt.Println("Input your id")
	fmt.Scanln(&id)
	return id, port
}

//和新节点建立连接
func newconnect(p string, k bool) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:"+p)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkErr(err)
	ConnMap[conn.RemoteAddr().String()] = Link{conn.RemoteAddr().String(), "127.0.0.1:" + port, conn}
	defer conn.Close()
	// 发送节点信息和map
	a := &Info{
		id, port,
	}
	b := &message{
		Kind:      "receive",
		Verifykey: k,
		Jlmessage: a,
		Peerinfo:  InfoMap,
		Talk:      "joined into the net",
	}
	jsonmake(b, conn)
	for {

		data := make([]byte, 10000)
		total, err := conn.Read(data)
		if err != nil {
			break
		}
		handle(data, total, conn)
	}

}

//建立与中心节点的连接
func centerlink(k string) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	defer conn.Close()
	ConnMap[conn.RemoteAddr().String()] = Link{conn.RemoteAddr().String(), "127.0.0.1:" + port, conn}
	//向中心节点发送自身信息
	if k == "receive" {
		//向中心节点发送自身信息
		a := &Info{
			id, port,
		}
		b := &message{
			Kind:      "newconnect",
			Jlmessage: a,
		}
		fmt.Println("d")
		jsonmake(b, conn)
	}
	for {

		data := make([]byte, 10000)
		total, err := conn.Read(data)
		if err != nil {
			break
		}
		handle(data, total, conn)
	}
}

//消息分类处理
func handle(data []byte, total int, conn *net.TCPConn) {
	v := &message{}
	err := json.Unmarshal(data[:total], v)
	checkErr(err)
	switch v.Kind {
	case "link":
		go centerlink(v.Kind)
	case "join":
		//连接到中心
		//发送请求
		//改变response的值
		//修改本地连接池和拓扑表
		if v.Verifykey == true {
			//来自客户端的消息
			fmt.Println(v.Jlmessage.Id + " 申请加入")
			tcpConn = ConnMap["127.0.0.1:10000"].Conninfo
			tcpConn.Write(data[:total])
			<-ch
			if response == true {
				fmt.Println("passed")
				//是否为第一个节点
				if v.Jlmessage.Port != port {
					//转发这个加入申请
					v.Verifykey = false
					b, _ := json.Marshal(v)
					for _, conn := range ConnMap {
						if conn.Remoteaddr != "127.0.0.1:10000" {
							conn.Conninfo.Write(b)
						}

					}
					go newconnect(v.Jlmessage.Port, true)
				} else {
					conn = ConnMap["127.0.0.1:10000"].Conninfo
					//向中心节点发送自身信息
					a := &Info{
						id, port,
					}
					b := &message{
						Kind:      "newconnect",
						Jlmessage: a,
					}
					jsonmake(b, conn)
				}
			} else {
				fmt.Println("nothing")
			}
		} else {
			InfoMap[v.Jlmessage.Port] = Info{v.Jlmessage.Id, v.Jlmessage.Port}
			fmt.Println(v.Jlmessage.Id + " joined the web and map updated")
			go newconnect(v.Jlmessage.Port, v.Verifykey)
		}
	case "receive":
		if v.Verifykey == true {
			go centerlink(v.Kind)
			fmt.Println(v.Talk)
			c, _ := json.Marshal(v.Peerinfo)
			err := json.Unmarshal(c, &InfoMap)
			ConnMap["127.0.0.1:"+v.Jlmessage.Port] = Link{"127.0.0.1:" + v.Jlmessage.Port, "127.0.0.1:" + port, conn}
			fmt.Println("get the map")
			checkErr(err)
		} else {
			//来自转发的节点的连接申请
			ConnMap["127.0.0.1:"+v.Jlmessage.Port] = Link{"127.0.0.1:" + v.Jlmessage.Port, "127.0.0.1:" + port, conn}

		}
	case "permission":
		response = v.Verifykey
		ch <- true
	case "clock":
		fmt.Println("checked")
		tcpConn = ConnMap["127.0.0.1:10000"].Conninfo
		b := &message{
			Kind:      "clock",
			Verifykey: true,
		}
		jsonmake(b, tcpConn)
	case "leave":
		if v.Verifykey == true {
			fmt.Println(v.Jlmessage.Id + " 申请离开")
			tcpConn = ConnMap["127.0.0.1:10000"].Conninfo
			tcpConn.Write(data[:total])
			<-ch
			if response == true {
				delete(InfoMap, v.Jlmessage.Port)
				delete(ConnMap, "127.0.0.1:"+v.Jlmessage.Port)
				fmt.Println(v.Jlmessage.Id + " leaved the net and map updated")
				v.Verifykey = false
				c, _ := json.Marshal(v)
				for _, conn := range ConnMap {
					if conn.Conninfo.RemoteAddr().String() != "127.0.0.1:10000" {
						//不向数据输入的客户端发送消息
						conn.Conninfo.Write(c)
					}
				}
			} else {
				fmt.Println("request refused")
			}
		} else {
			delete(InfoMap, v.Jlmessage.Port)
			delete(ConnMap, "127.0.0.1:"+v.Jlmessage.Port)
			fmt.Println(v.Jlmessage.Id + " leaved the net and map updated")
		}
	case "msg":
		fmt.Println(v.Talk)
	case "onchain":
		fmt.Println("上链消息")
	case "test":
		fmt.Println("checked")
	case "cleave":
		fmt.Println(v.Jlmessage.Id + "leaved")
		delete(InfoMap, v.Jlmessage.Port)
		delete(ConnMap, "127.0.0.1:"+v.Jlmessage.Port)
	}

}

//监听连接通道
func listen(tcpConn *net.TCPConn) {
	ipStr := tcpConn.RemoteAddr().String()
	// 中断处理
	defer func() {
		fmt.Println("disconnected :" + ipStr)
		tcpConn.Close()
	}()
	//循环监听
	for {
		data := make([]byte, 10000)
		total, err := tcpConn.Read(data)
		if err != nil {
			break
		}
		handle(data, total, tcpConn)
	}
}

//main
func main() {
	//输入节点信息
	id, port = input()

	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:"+port)
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	ConnMap = make(map[string]Link)
	//建立自身的map表格并初始化，以port为索引
	InfoMap = map[string]Info{
		port: Info{
			id,
			port,
		},
	}
	//开始监听消息
	for {
		tcpConn, _ := tcpListener.AcceptTCP()
		defer tcpConn.Close()
		// fmt.Println("连接的客服端信息:", tcpConn.RemoteAddr().String())
		//消息处理
		go listen(tcpConn)
	}
}
