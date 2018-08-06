# websocket
implement websocket in go and use the chating-room as practice

使用说明：

（1）go install mywebsocket (注意mywebsocket目录要在环境变量的目录下，新建环境变量或者直接放在原来go的库目录下src/)

（2）包内函数使用说明： 
     
     1、构造函数：NewWsocket(conn net.Conn) *Wsocket
     
     2、websocket握手函数：(ws *Wsocket)ShakeHand(headers map[string]string)
     
     3、解析帧函数：(ws *Wsocket) ParseFrame() []byte
     
     4、构造帧函数：(ws *Wsocket)BuildFrame(data []byte) []byte
     
     5、解析http报文函数：(ws *Wsocket)ParseHttp(content string) map[string]string
     

（3）运行服务端，go run server.go

（4）运行客户端，用浏览器打开client_online.html就行了


websocket聊天室主要是仿造：https://github.com/zhenbianshu/websocket

客户端完全是使用他的，然后服务端就自己写

websocket实现闲谈：

阅读资料：   http://www.infoq.com/cn/articles/deep-in-websocket-protocol

            https://github.com/abbshr/abbshr.github.io/issues/22
	    
            https://github.com/zhangkaitao/websocket-protocol/wiki/5.%E6%95%B0%E6%8D%AE%E5%B8%A7
	    
            http://www.cnblogs.com/yjf512/archive/2013/02/18/2915171.html
	    
            

实现需要解决的问题：

1、握手算法，对客户端发来的sec-websocket-key进行如下操作

base64_encode(sha1($key . "258EAFA5-E914-47DA-95CA-C5AB0DC85B11", true));

2、解析帧：需要对内容进行掩码解析

	//假设我们发送的"Payload data"以变量`data`表示，字节（byte）数为len;
	
	//masking_key为4byte的mask掩码组成的数组
	
	//offset：跳过的字节数
	
	for (var i = 0; i < len; i++) {
	
        var j = i % 4;
	
        data[offset + i] ^= masking_key[j];
	
    }
    
（1）没有分片：FIN为1，操作码非0的帧

（2）分片：

	a. FIN为0，操作码非0的帧
	
	b. FIN为0，操作码为0的帧
	
	c. FIN为0，操作码为0的帧
	
	d. …………..
	
	e. FIN为1，操作码为0的帧
    
        Note1：消息的分片必须由发送者按给定的顺序发送给接收者。
	
        Note2：控制帧禁止分片
	
        Note3：接受者不必按顺序缓存整个frame来处理
	
	
解析规则：

第一个字节：

	1bit: frame-fin，x0表示该message后续还有frame；x1表示是message的最后一个frame
	
	3bit: 分别是frame-rsv1、frame-rsv2和frame-rsv3，通常都是x0
	
	4bit: frame-opcode，x0表示是延续frame；x1表示文本frame；x2表示二进制frame；x3-7保留给非控制frame；x8表示关 闭连接；x9表示ping；xA表示pong；xB-F保留给控制frame
	
第二个字节：

	1bit: Mask，1表示该frame包含掩码；0，表示无掩码
	
	（在掩码字节段前可能还有2个或8个字节是描述负载长度的）
	
	7bit、7bit+2byte、7bit+8byte: 7bit取整数值，若在0-125之间，则是负载数据长度；若是126表示，后两个byte取无符号16位整数值，是负载长度；127表示后8个 byte，取64位无符号整数值，是负载长度

（没有负载长度的话）

第三到六个字节：

	这里假定负载长度在0-125之间，并且Mask为1，则这4个字节是掩码
	
第七到最后个字节：

长度是上面取出的负载长度，包括扩展数据和应用数据两部分，通常没有扩展数据；若Mask为1，则此数据需要解码，解码规则为1-4byte掩码循环和数据byte做异或操作。


基于websocket协议升级的攻击:

掩码：

    掩码键是由客户端随机选择的32位值。当准备一个掩码的帧时，客户端必须从允许的32位值集合中选择一个新的掩码键。掩码键需要是不可预测的；因此，掩码键必须来自一个强大的熵源，且用于给定帧的掩码键必须不容易被服务器/代理预测用于后续帧的掩码键。
    
    主要是用于解决协议转换（upgrade）引起的漏洞（污染代理服务器）

精心构建的报文

Client → Server:

POST /path/of/attackers/choice HTTP/1.1 Host: host-of-attackers-choice.com Sec-WebSocket-Key: <connection-key>

Server → Client:

HTTP/1.1 200 OK

Sec-WebSocket-Accept: <connection-key>

个人理解：

首先要知道代理服务器怎么样才会把资源缓存起来？当请求资源的http报文中的host与响应请求的http报文中的host相同，代理服务器就会把资源缓存起来了。

然后攻击代理服务器的漏洞是在http协议upgrade成websocket协议的那里（如果没有掩码）：

攻击者与邪恶服务器进行websocket连接时，首先是发一个upgrade的http报文，然后邪恶服务器响应回一个成功upgrade的http报文，在这里代理服务器不知道它之后就会通过websocket连接，只是把它当作一次http会话。

接着，攻击者通过刚才建立的websocket向邪恶服务器发送数据，数据就是楼主说的（精心构造的HTTP格式的文本），通过代理服务器时，代理服务器就把它当作了http请求报文，其中请求的url是正确资源的url，host是正确服务器的host，但是，实际上是通过websocket发给了邪恶服务器。

再者，邪恶服务器接收到这个websocket报文之后，只需要向攻击者发回一个http应答报文（报文内容是邪恶资源，其中host被伪造为正确服务器的host）。

最后，代理服务器接收到这个应答报文之后，由于应答报文的host跟之前的请求报文的host一样，所以缓存了这个邪恶资源（缓存记录的索引为正确资源url和正确服务器host，值为邪恶资源）。

当受害者以正确资源的url和正确服务器的host去访问的时候，代理服务器自然就会返回邪恶资源了。
