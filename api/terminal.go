package api

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

const END_OF_TRANSMISSION = "\u0004"

/*
	1 web container terminal
	2 20分钟无输入要退出bash进程
	3 需要支持心跳，不然websocket过腾讯云lb只能持续2分钟
	4 程序ctrl c之后，断开所有容器的bash进程 //未能成功
*/
type WebTerminal struct {
	conn     *websocket.Conn
	size     chan *remotecommand.TerminalSize
	timeout  time.Duration
	readNum  chan int
	canClose bool
	err      error
}

func init() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	go func() {
		<-ch
		{
			fmt.Println(errorProcessinterrupt)
			//cancel()
		}
	}()
}

//心跳检查的意义在于生产环境是lb  nginx ，有代理超时设置
//本地调试不会自动断开
const healthCheck = "[%heart_check%]"
const errorTimeOut = "\033[31m20 min 无输入 close\033[0m"
const errorProcessinterrupt = "\033[31m Process interrupt close\033[0m"

// NewWebTerminal web terminal的实现
func NewWebTerminal(conn *websocket.Conn, w, h uint16) *WebTerminal {
	term := &WebTerminal{
		conn:    conn,
		size:    make(chan *remotecommand.TerminalSize, 1),
		timeout: time.Minute * 20, //20分钟用户无输入发送eof
		readNum: make(chan int, 1),
	}
	term.size <- &remotecommand.TerminalSize{Width: w, Height: h}
	go func() {
		for {
			term.watchRead()
			if term.err != nil {
				log.Println("term.watchRead break")
				break
			}
		}
	}()

	return term
}

// 模拟stdout，stderr
func (a *WebTerminal) Write(p []byte) (n int, err error) {
	err = a.conn.WriteMessage(1, p)
	return len(p), err
}

// 模拟stdin
func (a *WebTerminal) Read(p []byte) (n int, err error) {
	if a.canClose {
		a.conn.WriteMessage(1, []byte(errorTimeOut))
		return 0, errors.New(errorTimeOut)
	}
	t, msg, err := a.conn.ReadMessage()
	defer func() {
		a.err = err
		a.readNum <- n
	}()
	//收到前端close信号之后 返回错误
	if t == websocket.CloseMessage {
		return 0, errors.New("websocket CloseMessage 8")
	}
	//前端心跳
	if string(msg) == healthCheck {
		return 0, nil
	}
	//复制k8s的代码,发送exit会eof
	if err != nil {
		n = copy(p, END_OF_TRANSMISSION)
		return n, err
	}
	n = copy(p, msg)
	return
}

func (a *WebTerminal) watchRead() {
	tf := time.After(a.timeout)
	select {
	case <-tf:
		a.canClose = true
	case <-a.readNum:
		return
	}
}

/*
// Next returns the new terminal size after the terminal has been resized. It returns nil when
// monitoring has been stopped.
注意 这里是设置终端大小的实现，终端字符个数是前端计算 传入后端的如果返回nil证明已经设置完成，不然k8s接口会频繁监听设置大小，导致终端卡顿
因此让k8s调用一次 然后关闭chan 返回nil
*/
func (a *WebTerminal) Next() *remotecommand.TerminalSize {
	if v, ok := <-a.size; ok {
		defer close(a.size)
		return v
	} else {
		return nil
	}
}
