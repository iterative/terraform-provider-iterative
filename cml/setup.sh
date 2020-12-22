#/bin/sh

DEBIAN_FRONTEND=noninteractive
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes

sudo apt update
sudo curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh && \
sudo usermod -aG docker ubuntu
sudo setfacl --modify user:ubuntu:rw /var/run/docker.sock

curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform

curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
sudo apt update && sudo apt-get install -y nodejs

sudo apt install -y ubuntu-drivers-common git
sudo ubuntu-drivers autoinstall 
curl -s -L https://nvidia.GitHub.io/nvidia-docker/gpgkey | sudo apt-key add - && \
curl -s -L https://nvidia.GitHub.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt update && sudo apt install -y nvidia-container-toolkit

sudo npm install -g git+https://github.com/iterative/cml.git#cml-runner
