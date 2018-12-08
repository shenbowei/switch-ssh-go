# switch-ssh-go
A packaged SSH library for switches (huawei,h3c,cisco).
A session pool is implemented to avoid repeated connection devices 
and automatically clear sessions that are not used for 10 minutes.

## Installation

```text
$ go get github.com/shenbowei/switch-ssh-go
```

## Basic Usage

### In Code

switch-ssh-go implemented a connection pool to save the session, 
and each session verifies its availability before executing the commands,
so you can call the following method repeatedly (not repeatedly connecting the device).


```go
//get the switch brand(vendor), include h3c,huawei and cisco
brand, err := ssh.GetSSHBrand(user, password, ipPort)

//run the cmds in the switch, and get the execution results
result, err := ssh.RunCommands(user, password, ipPort, cmds...)

//run the cmds in the switch with the device brand(the first connection will be faster), and get the execution results
result, err := ssh.RunCommandsWithBrand(user, password, ipPort, ssh.CISCO, cmds...)
```

### example

```go
package main

import (
    "fmt"
    "github.com/shenbowei/switch-ssh-go"
)

func main() {
    user := "your device ssh name"
    password := "your device ssh password"
    ipPort := "ip:22"

    //get the switch brand(vendor), include h3c,huawei and cisco
    brand, err := ssh.GetSSHBrand(user, password, ipPort)
    if err != nil {
        fmt.Println("GetSSHBrand err:\n", err.Error())
    }
    fmt.Println("Device brand is:\n", brand)

    //run the cmds in the switch, and get the execution results
    cmds := make([]string, 0)
    cmds = append(cmds, "dis clock")
    cmds = append(cmds, "dis vlan")
    result, err := ssh.RunCommands(user, password, ipPort, cmds...)
    if err != nil {
        fmt.Println("RunCommands err:\n", err.Error())
    }
    fmt.Println("RunCommands result:\n", result)
}

```

## Licenses

switch-ssh-go is released under the MIT License. 