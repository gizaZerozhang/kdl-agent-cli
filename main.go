// kdl-agent 调用快代理 Agent Gateway 的查询与授权操作。
package main

import (
	"os"

	"github.com/gizaZerozhang/kdl-agent-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
