package ssh

import (
	"testing"
	"time"
    "fmt"
)

func TestSSHRunner(t *testing.T) {

	user := ""
	password := ""
	ipPort := "ip:22"
	cmds := make([]string, 0)
	cmds = append(cmds, "dis clock")
	cmds = append(cmds, "dis clock")
	cmds = append(cmds, "dis clock")

    Debug = false
	result, err := RunCommandsWithBrand(user, password, ipPort, H3C, cmds...)
	if err != nil {
		LogError("RunCommands err:%s", err.Error())
	}
	fmt.Printf("RunCommands result:\n%s", result)

	time.Sleep(11 * time.Second)

	result2, err := RunCommandsWithBrand(user, password, ipPort, H3C, cmds...)
	if err != nil {
		LogError("RunCommands err:%s", err.Error())
	}
    fmt.Printf("RunCommands result2:\n%s", result2)
	time.Sleep(time.Second)
}

func TestSSHRunnerMultiple(t *testing.T) {
	user := ""
	password := ""
	ipPort := "ip:22"
	cmds := make([]string, 0)
	cmds = append(cmds, "dis clock")
	cmds = append(cmds, "dis vlan")

	for i := 0; i < 3; i++ {
		go func(i int) {
			result, err := RunCommands(user, password, ipPort, cmds...)
			if err != nil {
				LogError("RunCommands<%d> err:%s", i, err.Error())
			}
			LogDebug("RunCommands<%d> result:\n%s", i, result)
		}(i)
	}

	time.Sleep(60 * time.Second)
}

func TestGetSSHBrand(t *testing.T) {

	user := ""
	password := ""
	ipPorts := make([]string, 0)
	ipPorts = append(ipPorts, "ip:22") //huawei
	ipPorts = append(ipPorts, "ip:22") //h3c
	ipPorts = append(ipPorts, "ip:22") //cisco_nexus
	ipPorts = append(ipPorts, "ip:22") //cisco

	for _, ipPort := range ipPorts {
		brand, err := GetSSHBrand(user, password, ipPort)
		if err != nil {
			LogError("GetSSHBrand<%s> err:%s", ipPort, err.Error())
			return
		}
		LogDebug("GetSSHBrand <%s, %s>", ipPort, brand)

		time.Sleep(time.Second)
	}
}
