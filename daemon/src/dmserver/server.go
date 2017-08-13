package dmserver

import (
	"bufio"
	"colorlog"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

// 服务器状态码
// 已经关闭
const (
	SERVER_STATUS_CLOSED = iota
	SERVER_STATUS_RUNNING
	SERVER_STATUS_STATING
)

func (s *ServerRun) Close() {
	colorlog.LogPrint("Closing server...")
	var execConf ExecConf
	if server, ok := serverSaved[s.ID]; ok {
		var err error
		execConf, err = server.loadExecutableConfig()
		if err != nil {
			colorlog.ErrorPrint(err)
			return
		}
	}
	colorlog.LogPrint("Closing command is" + execConf.StoppedServerCommand)
	go time.AfterFunc(20*time.Second, func() {
		// 杀死进程组.
		colorlog.PointPrint("Timeout,Kill them.")
		if serverSaved[s.ID].Status != 0 {
			if s.Cmd.Process != nil {
				syscall.Kill(s.Cmd.Process.Pid, syscall.SIGKILL)
			}
		}
	})
	s.inputLine(execConf.StoppedServerCommand)
}

func (server *ServerLocal) Start() error {

	server.EnvPrepare()
	colorlog.LogPrint("Done.")
	execConf, err0 := server.loadExecutableConfig()
	if err0 != nil {
		return err0
	}
	commandArgs := []string{
		"-uid=" + strconv.Itoa(config.DaemonServer.UserId),
		"-cmd=" + execConf.Command,
		"-sid=" + strconv.Itoa(server.ID),
	}
	if execConf.ProcDir {
		commandArgs = append(commandArgs, "-proc")
	}
	cmd := exec.Command("./server", commandArgs...)
	//#########Testing###########
	stdoutPipe, err := cmd.StdoutPipe()

	if err != nil {
		return err
	}

	stdinPipe, err2 := cmd.StdinPipe()
	if err2 != nil {
		return err2
	}
	servers[server.ID] = &ServerRun{
		server.ID,
		[]string{},
		cmd,
		make([][]byte, 50),
		&stdinPipe,
		&stdoutPipe,
	}
	err3 := cmd.Start()
	server.Status = SERVER_STATUS_STATING
	if err3 != nil {
		return err3
	}
	start, join, left := execConf.getRegexps()
	cmdCgroup := exec.Command("/bin/bash",
		"../cgroup/cg.sh",
		"cg",
		"run",
		"server"+strconv.Itoa(server.ID),
		strconv.Itoa(cmd.Process.Pid))
	cmdCgroup.Env = os.Environ()
	output, err4 := cmdCgroup.CombinedOutput()
	if err4 != nil {
		colorlog.ErrorPrint(errors.New("Error with init cgroups:" + err4.Error()))
		colorlog.LogPrint("Reaseon:" + string(output))
		colorlog.PromptPrint("This server's source may not valid")
	}
	go servers[server.ID].ProcessOutput(start, join, left) // 将三个参数传递
	return nil
}

func (s *ServerRun) ProcessOutput(start, join, left *regexp.Regexp) {
	fmt.Println(s.Cmd.Process.Pid)
	buf := bufio.NewReader(*s.StdoutPipe)
	if _, ok := serverSaved[s.ID]; !ok {
		delete(servers, s.ID)
		return
	}
	go s.getServerStopped()
	for {
		if serverSaved[s.ID].Status == 0 {
			break
		}
		line, err := buf.ReadBytes('\n') //以'\n'为结束符读入一行
		if err != nil || io.EOF == err {
			break
		}
		fmt.Printf("%s", line)
		s.BufLog = append(s.BufLog[1:], line)
		s.processOutputLine(string(line), start, join, left) // string对与正则更加友好吧
		//s.ToOutput.IsOutput = true
		if isOut, to := IsOutput(s.ID); isOut {
			// 向ws客户端输出.
			to.WriteMessage(websocket.TextMessage, line)
		}
	}
	colorlog.LogPrint("Break for loop,server stopped or EOF. ")
	//delete(serverSaved,s.ID)

}

// 删除服务器
func (server *ServerLocal) Delete() {

	if server.Status == SERVER_STATUS_RUNNING {
		servers[server.ID].Close()
	}
	// 如果服务器仍然开启则先关闭服务器。
	// 成功关闭后，请Golang拆迁队清理违章建筑
	nowPath, _ := filepath.Abs(".")
	serverRunPath := filepath.Clean(nowPath + "/../servers/server" + strconv.Itoa(server.ID))
	os.RemoveAll(serverRunPath)
	// 清理服务器所占的储存空间
	// 违章搭建搞定以后，把这个记账本的东东也删掉
	// go这个切片是[,)左闭右开的区间，应该这么写吧~
	delete(serverSaved, server.ID)
	// 保存服务器信息。
	saveServerInfo()
}

func GetServerSaved() map[int]*ServerLocal {
	return serverSaved
}

func (s *ServerRun) getServerStopped() {
	s.Cmd.Wait()
	serverSaved[s.ID].Status = 0
	delete(serverSaved, s.ID)
	colorlog.PointPrint("Server Stopped")
}

// 获取那些正则表达式
func (e *ExecConf) getRegexps() (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp) {
	startReg, err := regexp.Compile(e.StartServerRegexp)
	if err != nil {
		colorlog.ErrorPrint(err)
		startReg = regexp.MustCompile("Done \\(.+s\\)!") // 用户自己的表达式骚写时,打印错误信息并使用系统默认表达式(1.7.2 spigot)
	}
	joinReg, err2 := regexp.Compile(e.NewPlayerJoinRegexp)
	if err2 != nil {
		colorlog.ErrorPrint(err2)
		joinReg = regexp.MustCompile("(\\w+)\\[.+\\] logged in")
	}
	exitReg, err3 := regexp.Compile(e.PlayExitRegexp)
	if err3 != nil {
		colorlog.ErrorPrint(err3)
		exitReg = regexp.MustCompile("(\\w+) left the game")
	}
	return startReg, joinReg, exitReg
}
