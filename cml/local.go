package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

func main() {
	data := make(map[string]string)
	data["AWS_SECRET_ACCESS_KEY"] = os.Getenv("AWS_SECRET_ACCESS_KEY")
	data["AWS_ACCESS_KEY_ID"] = "slkdksd+sldksldks"
	data["AZURE_CLIENT_ID"] = "++++++++"
	data["AZURE_CLIENT_SECRET"] = os.Getenv("AZURE_CLIENT_SECRET")
	data["AZURE_SUBSCRIPTION_ID"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
	data["AZURE_TENANT_ID"] = os.Getenv("AZURE_TENANT_ID")

	tmpl, _ := template.New("deploy").Parse(`#!/bin/bash
	++++
export AWS_SECRET_ACCESS_KEY={{.AWS_SECRET_ACCESS_KEY}}
export AWS_ACCESS_KEY_ID={{.AWS_ACCESS_KEY_ID}}
export AZURE_CLIENT_ID={{.AZURE_CLIENT_ID}}
export AZURE_CLIENT_SECRET={{.AZURE_CLIENT_SECRET}}
export AZURE_SUBSCRIPTION_ID={{.AZURE_SUBSCRIPTION_ID}}
export AZURE_TENANT_ID={{.AZURE_TENANT_ID}}
sleep 10
`)
	var customDataBuffer bytes.Buffer
	_ = tmpl.Execute(&customDataBuffer, data)
	fmt.Println(customDataBuffer.String())
	fmt.Println(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	fmt.Println(data["AZURE_CLIENT_ID"])
}
