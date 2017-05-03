//中心节点
//功能：
//1.定时向各服务器发送确认信息
//2.在节点加入时进行确认，本地保存了一个map
//3.备份表单信息
//4.心跳检测网络完整性

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
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
	Port       string
	Remoteaddr string
	Localaddr  string
	Conninfo   *net.TCPConn
}

//全局变量定义
var ok bool                 //中心节点是否通过
var port, id string         //本地ip
var InfoMap map[string]Info //本地拓扑
var ConnMap map[string]Link //连接池
var tcpConn *net.TCPConn

//错误检查函数
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func jsonmake(b *message, conn *net.TCPConn) {
	c, err := json.Marshal(b)
	if err != nil {
		fmt.Println("err:", err)
	}
	conn.Write(c)
}

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

func heartbeat() {

	b := &message{
		Kind: "test",
	}

	c, err := json.Marshal(b)
	if err != nil {
		fmt.Println("err:", err)
	}

	for {
		time.Sleep(3 * time.Second) //延时三秒
		for _, conn := range ConnMap {
			_, err := conn.Conninfo.Write(c)
			if err != nil {
				//广播ip地址，从map中删除
				fmt.Println("有连接断开了,ip为" + conn.Remoteaddr)
				// delete(InfoMap, conn.Remoteaddr)
				delete(InfoMap, conn.Port)
				delete(ConnMap, conn.Remoteaddr)
				fmt.Println(InfoMap)
				broadcast(InfoMap[conn.Port].Id, conn.Port)
				continue
			}
		}

	}
}

func broadcast(i string, p string) {
	fmt.Println("dealing")
	for _, conn := range ConnMap {
		a := &Info{
			i,
			p,
		}
		b := message{
			Kind:      "cleave",
			Verifykey: true,
			Jlmessage: a,
		}
		c, _ := json.Marshal(b)
		_, err := conn.Conninfo.Write(c)
		if err != nil {
			continue
		}

	}
	fmt.Println("done")

}

func handle(data []byte, total int, tcpConn *net.TCPConn) {
	v := &message{}
	err := json.Unmarshal(data[:total], v)
	checkErr(err)
	switch v.Kind {
	case "newconnect":
		ConnMap["127.0.0.1:"+v.Jlmessage.Port] = Link{v.Jlmessage.Port, "127.0.0.1:" + v.Jlmessage.Port, "127.0.0.1:10000", tcpConn}
		fmt.Println("connmap update")
	case "join":
		fmt.Println("join request received")
		_, ok := InfoMap[v.Jlmessage.Port]
		if ok == true {
			fmt.Println("request refused")
			b := &message{
				Kind:      "permission",
				Verifykey: false,
			}
			jsonmake(b, tcpConn)
			fmt.Println(InfoMap)
		} else {
			fmt.Println("request passed")
			InfoMap[v.Jlmessage.Port] = Info{v.Jlmessage.Id, v.Jlmessage.Port}
			b := &message{
				Kind:      "permission",
				Verifykey: true,
			}
			jsonmake(b, tcpConn)
			fmt.Println(InfoMap)
		}
	case "leave":
		fmt.Println("leave request received")
		_, ok := InfoMap[v.Jlmessage.Port]
		if ok == true {
			fmt.Println("request passed")
			delete(InfoMap, v.Jlmessage.Port)
			delete(ConnMap, "127.0.0.1:"+v.Jlmessage.Port)
			b := &message{
				Kind:      "permission",
				Verifykey: true,
			}
			jsonmake(b, tcpConn)
			fmt.Println(InfoMap)
		} else {
			fmt.Println("request refused")
			b := &message{
				Kind:      "permission",
				Verifykey: false,
			}
			jsonmake(b, tcpConn)
			fmt.Println(InfoMap)
		}
	}
}

//main
func main() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:10000")
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	fmt.Println("start listen")
	ConnMap = make(map[string]Link)
	InfoMap = make(map[string]Info)
	go heartbeat()
	for {

		tcpConn, _ := tcpListener.AcceptTCP()
		defer tcpConn.Close()

		fmt.Println("连接的客服端信息:", tcpConn.RemoteAddr().String())
		go listen(tcpConn)
	}
}
