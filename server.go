package main

import (
	"fmt"
	"net"
	"encoding/json"
	"mywebsocket"
)
type socket_peer map[net.Conn]map[string]interface{} //存用户信息

//构造前端需要的json格式以及向每个用户广播
func (sp socket_peer)buildMessage(data map[string]interface{}, wsconn *mywebsocket.Wsocket) {
	msg_type := data["type"]
	msg_content := data["content"]
	msg := make(map[string]interface{})
	tmpmap := make(map[string]interface{})
	switch msg_type {
		case "login":
			tmpmap["uname"] = msg_content
			sp[wsconn.Conn] = tmpmap
			msg["type"] = "login"
			msg["content"] = msg_content
			msg["user_list"] = sp.get_sp_names()
		case "logout":
			msg["type"] = "logout"
			msg["content"] = msg_content
			msg["user_list"] = sp.get_sp_names()
		case "user":
			uname := sp[wsconn.Conn]["uname"]				
			msg["type"] = "user"
			msg["from"] = uname
			msg["content"] = msg_content
	}

	encoded_msg, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
	}

	frame := wsconn.BuildFrame(encoded_msg)
	sp.broadcast(frame)
}

//获取当前用户列表
func (sp socket_peer)get_sp_names() []interface{} {
	result := make([]interface{}, 0)
	for _,value := range sp {
		result = append(result, value["uname"])
	}
	return result
}

//向每个websocket用户广播
func (sp socket_peer)broadcast(data []byte) {
	for socket,_ := range sp {
		socket.Write(data)
	}
}

//构造断开连接的json信息，并把断开连接的用户从用户列表中删除
func (sp socket_peer)disconnect(wsconn *mywebsocket.Wsocket) map[string]interface{} {
	msg := make(map[string]interface{})
	msg["type"] = "logout"
	msg["content"] = sp[wsconn.Conn]["uname"]
	delete(sp, wsconn.Conn)
	return msg
}

//websocket握手，并发送握手成功的报文
func server_handshake(content string, wsconn *mywebsocket.Wsocket) {
	headers := wsconn.ParseHttp(string(content))
	wsconn.ShakeHand(headers)	
	msg := make(map[string]interface{})
	msg["type"] = "handshake"
	msg["content"] = "done"
	encoded_msg, err := json.Marshal(msg)//json编码
	if err != nil {
		fmt.Println(err)
	}
	frame := wsconn.BuildFrame(encoded_msg)
	wsconn.Conn.Write(frame)
}

func main() {
	tp, err := net.Listen("tcp", ":8000")
	if err != nil {
		fmt.Println(err)
	}
	sp := make(socket_peer)
	for {
		conn, err := tp.Accept()
		if err != nil {
			fmt.Println("Accept err:", err)
		}
		go sp.connect(conn)	//新建一个goroutine，不同socket并发操作
	}

}

func (sp socket_peer)connect(conn net.Conn) {
	wsconn := mywebsocket.NewWsocket(conn)
	content := make([]byte, 1024)
	_, rerr := conn.Read(content)
	if rerr != nil {
		fmt.Println(rerr)
	}
	server_handshake(string(content), wsconn)
	for {
		data := wsconn.ParseFrame()
		if len(data) < 9 { //帧大小小于9字节为断开连接的请求帧
			disconnect_msg := sp.disconnect(wsconn)
			sp.buildMessage(disconnect_msg, wsconn)
			break;
		}
		var tmp map[string]interface{}
		json.Unmarshal(data, &tmp)//json解码
		sp.buildMessage(tmp, wsconn)
		fmt.Println(tmp)
	}	
}
