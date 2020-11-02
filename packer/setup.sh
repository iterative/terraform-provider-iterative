#/bin/sh

echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes

curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh &&
  sudo usermod -aG docker \${USER}
sudo setfacl --modify user:\${USER}:rw /var/run/docker.sock

curl -s -L https://nvidia.GitHub.io/nvidia-docker/gpgkey | sudo apt-key add - &&
  curl -s -L https://nvidia.GitHub.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list &&
  sudo apt update && sudo apt install -y ubuntu-drivers-common &&
  sudo ubuntu-drivers autoinstall &&
  sudo apt install -y nvidia-container-toolkit
