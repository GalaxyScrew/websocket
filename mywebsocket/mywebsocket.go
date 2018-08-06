package mywebsocket

import (
	"fmt"
	"net"
	"strings"
	"crypto/sha1"
	"encoding/base64"
	"io"
)

//握手时生成Sec-WebSocket-Key需要的魔串
const AmazingStr = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"	

//作一个goroutine里的全局变量的作用
type Wsocket struct {
	Conn net.Conn
}

//构造函数
func NewWsocket(conn net.Conn) *Wsocket {
	return &Wsocket{conn}
}


/*
websocket握手（upgrade协议）
*/
func (ws *Wsocket)ShakeHand(headers map[string]string) {
	secWebsocketKey := headers["Sec-WebSocket-Key"]
	h := sha1.New()
	io.WriteString(h, secWebsocketKey + AmazingStr)
	secWebsocketAccept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	response := "HTTP/1.1 101 Switching Protocols\r\n"
	response = response + "Sec-WebSocket-Accept: " + secWebsocketAccept + "\r\n"
	response = response + "Connection: Upgrade\r\n"
	response = response + "Upgrade: websocket\r\n\r\n"

	_,err := ws.Conn.Write([]byte(response))

	if err != nil {
		fmt.Println(err)
	}
}


/*
读取和解析帧
（第一个字节）buffer[0] 为8x，x为非零值时，为一个非分片帧
帧解析规则是：
第一个字节：
	1bit: frame-fin，x0表示该message后续还有frame；x1表示是message的最后一个frame
	3bit: 分别是frame-rsv1、frame-rsv2和frame-rsv3，通常都是x0
	4bit: frame-opcode，x0表示是延续frame；x1表示文本frame；x2表示二进制frame；
			x3-7保留给非控制frame；x8表示关 闭连接；
			x9表示ping；xA表示pong；xB-F保留给控制frame
第二个字节：
	1bit: Mask，1表示该frame包含掩码；0，表示无掩码
	（在掩码字节段前可能还有2个或8个字节是描述负载长度的）
	7bit、7bit+2byte、7bit+8byte: 7bit取整数值，若在0-125之间，则是负载数据长度；
		若是126表示，后两个byte取无符号16位整数值，是负载长度；
		127表示后8个 byte，取64位无符号整数值，是负载长度
（没有负载长度的话）
第三到六个字节：
	这里假定负载长度在0-125之间，并且Mask为1，则这4个字节是掩码
第七到最后个字节：
 长度是上面取出的负载长度，包括扩展数据和应用数据两部分，通常没有扩展数据；
	若Mask为1，则此数据需要解码，解码规则为1-4byte掩码循环和数据byte做异或操作
*/
func (ws *Wsocket) ParseFrame() []byte {
	opcode := make([]byte, 1)
	ws.Conn.Read(opcode)
	FIN := opcode[0] >> 7

	payloadLen := make([]byte, 1)
	ws.Conn.Read(payloadLen)

	dlen := int(payloadLen[0]) & 127	//去掉掩码那一bit
	maskingkey := make([]byte, 4)
	if dlen == 126 {
		extendTwo := make([]byte, 2)
		ws.Conn.Read(extendTwo)
	} else if dlen == 127 {
		extendEight := make([]byte, 8)
		ws.Conn.Read(extendEight)
	}
	ws.Conn.Read(maskingkey)

	data := make([]byte, dlen)
	ws.Conn.Read(data)

	//进行掩码处理
	for i := 0; i < dlen; i++ {
		data[i] = data[i] ^ maskingkey[i % 4]
	}

	if FIN == 1 {
		return data
	}

	nextData := ws.ParseFrame() //分片时还要继续解析帧

	data =  append(data, nextData...)

	return data
}

//构建和发送帧,只构建操作码为1的非分片帧，服务端发往客户端的帧不需要掩码
func (ws *Wsocket)BuildFrame(data []byte) []byte{

	dlen := len(data)

	payloadLen := byte(0x00) | byte(dlen) //将掩码位置0

	result := []byte{0x81, payloadLen}

	result = append(result, data...)

	return result
}

//解析http报文
func (ws *Wsocket)ParseHttp(content string) map[string]string {
	headers := make(map[string]string, 10)
	lines := strings.Split(content, "\r\n")

	for _,line := range lines {
		if len(line) >= 0 {
			words := strings.Split(line, ":")
			if len(words) == 2 {
				headers[strings.Trim(words[0], " ")] = strings.Trim(words[1], " ")
			}
		}
	}
	return headers
}


