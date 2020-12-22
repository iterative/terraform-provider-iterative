//usr/bin/env go run $0 "$@"; exit
package main

import (
	"fmt"

	localexec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	c := terraform.NewResourceConfigRaw(map[string]interface{}{
		"command": "echo 'Hello world' > hello.txt",
	})
	p := localexec.Provisioner()
	if err := p.Apply(new(terraform.MockUIOutput), nil, c); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("nice")
	}
}
