#!/bin/bash

DEBIAN_FRONTEND=noninteractive
echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes
	 
curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform nodejs
sudo npm install -g git+https://github.com/iterative/cml.git#cml-runner

echo 'launching runner'

nohup cml-runner \
  --name furia4 \
  --workspace ~/runner \
  --labels tf \
  --idle-timeout 180 \
  --repo https://gitlab.com/DavidGOrtega/3_tensorboard \
  --token arszDpb3xtNdKaXmQ6vN < /dev/null > std.out 2> std.err &

sleep 5
echo 'Finished'
