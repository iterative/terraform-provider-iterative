package machine

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"terraform-provider-iterative/task/common"
)

func Script(script string, variables common.Variables, timeout time.Duration) string {
	var environment string
	for name, value := range variables.Enrich() {
		escaped := strings.ReplaceAll(value, `"`, `\"`) // FIXME: \" edge cases.
		environment += fmt.Sprintf("%s=\"%s\"\n", name, escaped)
	}

	return fmt.Sprintf(
		`#!/bin/bash
sudo mkdir --parents /tmp/tpi-task
chmod u=rwx,g=rwx,o=rwx /tmp/tpi-task

base64 --decode << END | sudo tee /usr/bin/tpi-task > /dev/null
%s
END
chmod u=rwx,g=rx,a=rx /usr/bin/tpi-task

base64 --decode << END | sudo tee /tmp/tpi-environment > /dev/null
%s
END
chmod u=rw,g=,o= /tmp/tpi-environment

while IFS= read -rd $'\0' variable; do
  export "$(perl -0777p -e 's/\\"/"/g;' -e 's/(.+?)="(.+)"/$1=$2/sg' <<< "$variable")"
done < <(perl -0777pe 's/\n*(.+?=".*?((?<!\\)"|\\\\"))\n*/$1\x00/sg' /tmp/tpi-environment)

sudo tee /etc/systemd/system/tpi-task.service > /dev/null <<END
[Unit]
  After=default.target
[Service]
  Type=simple
  ExecStart=/usr/bin/tpi-task
  ExecStop=/bin/bash -c 'systemctl is-system-running | grep stopping || tpi --stop'
  EnvironmentFile=/tmp/tpi-environment
  WorkingDirectory=/tmp/tpi-task
  TimeoutStartSec=%d
[Install]
  WantedBy=default.target
END

curl --location --remote-name https://github.com/iterative/terraform-provider-iterative/releases/latest/download/terraform-provider-iterative_linux_amd64
sudo mv terraform-provider-tpi* /usr/bin/tpi
sudo chmod u=rwx,g=rx,o=rx /usr/bin/tpi
sudo chown root:root /usr/bin/tpi

curl --remote-name https://downloads.rclone.org/rclone-current-linux-amd64.zip
python3 -m zipfile -e rclone-current-linux-amd64.zip .
sudo cp rclone-*-linux-amd64/rclone /usr/bin
sudo chmod u=rwx,g=rx,o=rx /usr/bin/rclone
sudo chown root:root /usr/bin/rclone
rm --recursive rclone-*-linux-amd64*

rclone copy "$RCLONE_REMOTE/data" /tmp/tpi-task

sudo systemctl daemon-reload
sudo systemctl enable tpi-task.service --now

TPI_MACHINE_IDENTITY="$(uuidgen)"
TPI_LOG_DIRECTORY="$(mktemp --directory)"

while sleep 5; do
  journalctl > "$TPI_LOG_DIRECTORY/machine-$TPI_MACHINE_IDENTITY"
  journalctl --unit tpi-task > "$TPI_LOG_DIRECTORY/task-$TPI_MACHINE_IDENTITY"
  rclone copy "$TPI_LOG_DIRECTORY" "$RCLONE_REMOTE/log"
done &

while sleep 10; do
  rclone copy /tmp/tpi-task "$RCLONE_REMOTE/data"
done &
`,
		base64.StdEncoding.EncodeToString([]byte(script)),
		base64.StdEncoding.EncodeToString([]byte(environment)),
		timeout/time.Second)
}
