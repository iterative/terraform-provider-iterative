#/bin/sh
PS4='tpi:setup.sh: '
set -x

export DEBIAN_FRONTEND=noninteractive
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes

sudo apt remove unattended-upgrades
systemctl disable apt-daily-upgrade.service
  
FILE=/var/log/cml_stack.log
if [ ! -f "$FILE" ]; then
  sudo add-apt-repository universe -y
  sudo add-apt-repository ppa:git-core/ppa -y
  sudo apt update && sudo apt-get install -y software-properties-common build-essential git acpid

  sudo curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
  sudo usermod -aG docker ubuntu
  sudo setfacl --modify user:ubuntu:rw /var/run/docker.sock

  sudo curl --max-time 10 --output /usr/bin/docker-credential-ecr-login https://amazon-ecr-credential-helper-releases.s3.us-east-2.amazonaws.com/0.5.0/linux-amd64/docker-credential-ecr-login
  sudo chmod a+x /usr/bin/docker-credential-ecr-login
    
  curl --max-time 10 --location https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.1.5/docker-credential-gcr_linux_amd64-2.1.5.tar.gz | sudo tar xz docker-credential-gcr
  sudo mv docker-credential-gcr /usr/bin/docker-credential-gcr

  curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
  sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
  sudo apt update && sudo apt-get install -y terraform

  curl -sL https://deb.nodesource.com/setup_16.x | sudo bash
  sudo apt update && sudo apt-get install -y nodejs

  sudo apt install -y ubuntu-drivers-common
  if ubuntu-drivers devices | grep -q NVIDIA; then
    sudo ubuntu-drivers install

    curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
    curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
    sudo apt update && sudo apt install -y nvidia-docker2
    sudo systemctl restart docker
  fi

  echo OK | sudo tee "$FILE"
fi
