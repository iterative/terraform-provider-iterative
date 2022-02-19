package machine

import (
	"encoding/base64"
	"fmt"
	"strconv"
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

	timeoutString := strconv.Itoa(int(timeout / time.Second))
	if timeout <= 0 {
		timeoutString = "infinity"
	}

	return fmt.Sprintf(
		`#!/bin/bash
sudo mkdir --parents /tmp/tpi-task
chmod u=rwx,g=rwx,o=rwx /tmp/tpi-task

base64 --decode << END | sudo tee /usr/bin/tpi-task > /dev/null
%s
END
chmod u=rwx,g=rx,a=rx /usr/bin/tpi-task

sudo tee /usr/bin/tpi-task-shutdown << 'END'
#!/bin/bash
sleep 20; while pgrep rclone > /dev/null; do sleep 1; done
if ! test -z "$CI"; then
  cml rerun-workflow
fi
(systemctl is-system-running | grep stopping) || tpi --stop;
END
chmod u=rwx,g=rx,o=rx /usr/bin/tpi-task-shutdown

base64 --decode << END | sudo tee /tmp/tpi-environment > /dev/null
%s
END
chmod u=rw,g=,o= /tmp/tpi-environment

while IFS= read -rd $'\0' variable; do
  export "$(perl -0777p -e 's/\\"/"/g;' -e 's/(.+?)="(.+)"/$1=$2/sg' <<< "$variable")"
done < <(perl -0777pe 's/\n*(.+?=".*?((?<!\\)"|\\\\"))\n*/$1\x00/sg' /tmp/tpi-environment)

TPI_MACHINE_IDENTITY="$(uuidgen)"
TPI_LOG_DIRECTORY="$(mktemp --directory)"
TPI_DATA_DIRECTORY="/tmp/tpi-task"

sudo tee /etc/systemd/system/tpi-task.service > /dev/null <<END
[Unit]
  After=default.target
[Service]
  Type=simple
  ExecStart=/usr/bin/tpi-task
  ExecStop=/bin/bash -c 'systemctl is-system-running | grep stopping || echo "{\\\\"result\\\\": \\\\"\$SERVICE_RESULT\\\\", \\\\"code\\\\": \\\\"\$EXIT_STATUS\\\\", \\\\"status\\\\": \\\\"\$EXIT_CODE\\\\"}" > "$TPI_LOG_DIRECTORY/status-$TPI_MACHINE_IDENTITY" && rclone copy "$TPI_LOG_DIRECTORY" "$RCLONE_REMOTE/reports"'
  ExecStopPost=/usr/bin/tpi-task-shutdown
  Environment=HOME=/root
  EnvironmentFile=/tmp/tpi-environment
  WorkingDirectory=/tmp/tpi-task
  TimeoutStartSec=%s
  TimeoutStopSec=infinity
[Install]
  WantedBy=default.target
END

curl --location --remote-name https://github.com/iterative/terraform-provider-iterative/releases/latest/download/terraform-provider-iterative_linux_amd64
sudo mv terraform-provider-iterative* /usr/bin/tpi
sudo chmod u=rwx,g=rx,o=rx /usr/bin/tpi
sudo chown root:root /usr/bin/tpi

curl --location --remote-name https://github.com/iterative/cml/releases/latest/download/cml-linux
chmod u=rwx,g=rx,o=rx cml-linux
sudo mv cml-linux /usr/bin/cml

extract_here(){
  if command -v unzip 2>&1 > /dev/null; then
    unzip "$1"
  elif command -v python3 2>&1 > /dev/null; then
    python3 -m zipfile -e "$1" .
  else
    python -m zipfile -e "$1" .
  fi
}

if ! command -v rclone 2>&1 > /dev/null; then
  curl --remote-name https://downloads.rclone.org/rclone-current-linux-amd64.zip
  extract_here rclone-current-linux-amd64.zip
  sudo cp rclone-*-linux-amd64/rclone /usr/bin
  sudo chmod u=rwx,g=rx,o=rx /usr/bin/rclone
  sudo chown root:root /usr/bin/rclone
  rm --recursive rclone-*-linux-amd64*
fi

rclone copy "$RCLONE_REMOTE/data" /tmp/tpi-task

sudo systemctl daemon-reload
sudo systemctl enable tpi-task.service --now

while sleep 5; do
  journalctl > "$TPI_LOG_DIRECTORY/machine-$TPI_MACHINE_IDENTITY"
  journalctl --unit tpi-task > "$TPI_LOG_DIRECTORY/task-$TPI_MACHINE_IDENTITY"
  NEW_TPI_LOG_DIRECTORY_HASH="$(md5sum "$TPI_LOG_DIRECTORY"/*)"
  if [[ "$NEW_TPI_LOG_DIRECTORY_HASH" != "$TPI_LOG_DIRECTORY_HASH" ]]; then
    TPI_LOG_DIRECTORY_HASH="$NEW_TPI_LOG_DIRECTORY_HASH"
    rclone sync "$TPI_LOG_DIRECTORY" "$RCLONE_REMOTE/reports"
  fi
done &

while sleep 10; do
  NEW_TPI_DATA_DIRECTORY_EPOCH="$(find "$TPI_DATA_DIRECTORY" -printf "%%T@\n" | sort | tail -1)"
  if [[ "$NEW_TPI_DATA_DIRECTORY_EPOCH" != "$TPI_DATA_DIRECTORY_EPOCH" ]]; then
    TPI_DATA_DIRECTORY_EPOCH="$NEW_TPI_DATA_DIRECTORY_EPOCH"
    rclone sync "$TPI_DATA_DIRECTORY" "$RCLONE_REMOTE/data"
  fi
done &
`,
		base64.StdEncoding.EncodeToString([]byte(script)),
		base64.StdEncoding.EncodeToString([]byte(environment)),
		timeoutString)
}
