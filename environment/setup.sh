#/bin/sh
FILE=/var/log/cml_stack.log
log_run() {
  printf "tpi:setup.sh: cmd --- %s ---\n" "$1"
  eval $1
}
if [ ! -f "$FILE" ]; then
  DEBIAN_FRONTEND=noninteractive
  echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes
  log_run "sudo apt remove unattended-upgrades"

  log_run "systemctl disable apt-daily-upgrade.service"

  log_run "sudo add-apt-repository universe -y"
  log_run "sudo add-apt-repository ppa:git-core/ppa -y"
  log_run "sudo apt update && sudo apt-get install -y software-properties-common build-essential git"

  log_run "sudo curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"
  log_run "sudo usermod -aG docker ubuntu"
  log_run "sudo setfacl --modify user:ubuntu:rw /var/run/docker.sock"

  log_run "curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -"
  log_run 'sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"'
  log_run "sudo apt update && sudo apt-get install -y terraform"

  log_run "curl -sL https://deb.nodesource.com/setup_16.x | sudo bash"
  log_run "sudo apt update && sudo apt-get install -y nodejs"

  log_run "sudo apt install -y ubuntu-drivers-common"
  log_run "sudo ubuntu-drivers autoinstall"

  log_run "sudo curl https://amazon-ecr-credential-helper-releases.s3.us-east-2.amazonaws.com/0.5.0/linux-amd64/docker-credential-ecr-login --output /usr/bin/docker-credential-ecr-login"
  log_run "sudo chmod 755 /usr/bin/docker-credential-ecr-login"

  log_run "curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -"
  log_run "curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list"
  log_run "sudo apt update && sudo apt install -y nvidia-docker2"
  log_run "sudo systemctl restart docker"

  echo OK | sudo tee "$FILE"
fi
