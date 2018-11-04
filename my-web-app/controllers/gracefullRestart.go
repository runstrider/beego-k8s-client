package controllers

//下方注释很重要，是我实践发现的。
//需要编译成bin文件运行，go run生成临时文件导致第一个进程运行完后被删除，后续的进程启动失败
import (
"os"
"fmt"
"flag"
"syscall"
"net"
"sync"
"os/exec"
"os/signal"
)

var (
	graceful = flag.Bool("graceful", false, "-graceful")
)

type Accepted struct {
	conn net.Conn
	err error
}

func firstBootListener() net.Listener{
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
	}

	return ln
}

func gracefulListener() net.Listener {
	ln, err := net.FileListener(os.NewFile(3, "graceful server"))
	if err != nil {
		fmt.Println(err)
	}

	return ln
}

// child take control
func forkAndRun(ln net.Listener){
	l := ln.(*net.TCPListener)
	newFile, _ := l.File()
	fmt.Println(newFile)

	cmd := exec.Command(os.Args[0], "-graceful")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.ExtraFiles = []*os.File{newFile}

	cmd.Start()

	os.Exit(0) //stop the Fater, and child take over by init(0).

	//注意这里的注释
	//ln.Close() //delete the father's listener, just allow child to listen.
	//Ever i thought that i should close the ln. But the fact is when i close the
	// ln, the child's newFile is nil. So i know they have a relation.
	// So, i can find. The father and the child can server the requset both. It all
	//depand on random.
}

func handleConnection(conn net.Conn, w *sync.WaitGroup){
	conn.Write([]byte("Hello"))
	conn.Close()
	w.Done()
}

// estblished http server
func listenAndServer(ln net.Listener, sig chan os.Signal, b bool){
	accept := make(chan Accepted, 1)

	var w sync.WaitGroup
	go func() {
		for {
			fmt.Println(b)
			conn, err := ln.Accept()
			if err == nil {
				accept <- Accepted{conn, err}
			}else {
				break
			}
		}
	}()

	for {
		select{
		case a := <- accept:
			if a.err == nil {
				w.Add(1)
				go handleConnection(a.conn, &w)
			}
		case _= <- sig:
			forkAndRun(ln)
		}
		w.Wait()
	}

}

func main(){
	flag.Parse()
	fmt.Printf("Give args : %t, pid: %d\n", *graceful, os.Getegid())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)

	var ln net.Listener

	if *graceful{
		ln = gracefulListener()
	} else {
		ln = firstBootListener()
	}

	listenAndServer(ln, c, *graceful)
}
