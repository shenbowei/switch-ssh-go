package ssh

import (
	"fmt"
	"strings"
	"time"
)

const (
	HUAWEI = "huawei"
	H3C    = "h3c"
	CISCO  = "cisco"
)

var IsLogDebug = true

/**
 * 外部调用的统一方法，完成获取会话（若不存在，则会创建连接和会话，并存放入缓存），执行指令的流程，返回执行结果
 * @param user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口, cmds 执行的指令(可以多个)
 * @return 执行的输出结果和执行错误
 * @author shenbowei
 */
func RunCommands(user, password, ipPort string, cmds ...string) (string, error) {
	sessionKey := user + "_" + password + "_" + ipPort
	sessionManager.LockSession(sessionKey)
	defer sessionManager.UnlockSession(sessionKey)

	sshSession, err := sessionManager.GetSession(user, password, ipPort, "")
	if err != nil {
		LogError("GetSession error:%s", err)
		return "", err
	}
	sshSession.WriteChannel(cmds...)
	result := sshSession.ReadChannelTiming(2 * time.Second)
	filteredResult := filterResult(result, cmds[0])
	return filteredResult, nil
}

/**
 * 外部调用的统一方法，完成获取会话（若不存在，则会创建连接和会话，并存放入缓存），执行指令的流程，返回执行结果
 * @param user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口, brand 交换机品牌（可为空）， cmds 执行的指令(可以多个)
 * @return 执行的输出结果和执行错误
 * @author shenbowei
 */
func RunCommandsWithBrand(user, password, ipPort, brand string, cmds ...string) (string, error) {
	sessionKey := user + "_" + password + "_" + ipPort
	sessionManager.LockSession(sessionKey)
	defer sessionManager.UnlockSession(sessionKey)

	sshSession, err := sessionManager.GetSession(user, password, ipPort, brand)
	if err != nil {
		LogError("GetSession error:%s", err)
		return "", err
	}
	sshSession.WriteChannel(cmds...)
	result := sshSession.ReadChannelTiming(2 * time.Second)
	filteredResult := filterResult(result, cmds[0])
	return filteredResult, nil
}

/**
 * 外部调用的统一方法，完成获取交换机的型号
 * @param user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return 设备品牌（huawei，h3c，cisco，""）和执行错误
 * @author shenbowei
 */
func GetSSHBrand(user, password, ipPort string) (string, error) {
	sessionKey := user + "_" + password + "_" + ipPort
	sessionManager.LockSession(sessionKey)
	defer sessionManager.UnlockSession(sessionKey)

	sshSession, err := sessionManager.GetSession(user, password, ipPort, "")
	if err != nil {
		LogError("GetSession error:%s", err)
		return "", err
	}
	return sshSession.GetSSHBrand(), nil
}

/**
 * 对交换机执行的结果进行过滤
 * @paramn result:返回的执行结果（可能包含脏数据）, firstCmd:执行的第一条指令
 * @return 过滤后的执行结果
 * @author shenbowei
 */
func filterResult(result, firstCmd string) string {
	//对结果进行处理，截取出指令后的部分
	filteredResult := ""
	resultArray := strings.Split(result, "\n")
	findCmd := false
	promptStr := ""
	for _, resultItem := range resultArray {
		resultItem = strings.Replace(resultItem, " \b", "", -1)
		if findCmd && (promptStr == "" || strings.Replace(resultItem, promptStr, "", -1) != "") {
			filteredResult += resultItem + "\n"
			continue
		}
		if strings.Contains(resultItem, firstCmd) {
			findCmd = true
			promptStr = resultItem[0:strings.Index(resultItem, firstCmd)]
			promptStr = strings.Replace(promptStr, "\r", "", -1)
			promptStr = strings.TrimSpace(promptStr)
			LogDebug("Find promptStr='%s'", promptStr)
			//将命令添加到结果中
			filteredResult += resultItem + "\n"
		}
	}
	if !findCmd {
		return result
	}
	return filteredResult
}

func LogDebug(format string, a ...interface{}) {
	if IsLogDebug {
		fmt.Println("[DEBUG]:" + fmt.Sprintf(format, a...))
	}
}

func LogError(format string, a ...interface{}) {
	fmt.Println("[ERROR]:" + fmt.Sprintf(format, a...))
}
