package iterative

import (
	"os"
	"testing"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)



const (
	expectedProvisionerCodeCloudInvalid string = `#!/bin/sh
export DEBIAN_FRONTEND=noninteractive



sudo npm install -g git+https://github.com/iterative/cml.git

sudo bash -c 'cat << EOF > /usr/bin/cml.sh
#!/bin/sh








cml-runner --name '10 value with "quotes" and spaces' --labels '16 value with "quotes" and spaces' --idle-timeout 11 --driver '15 value with "quotes" and spaces' --repo '14 value with "quotes" and spaces' --token '13 value with "quotes" and spaces' --tf_resource=eyJtb2RlIjoibWFuYWdlZCIsInR5cGUiOiJpdGVyYXRpdmVfY21sX3J1bm5lciIsIm5hbWUiOiJydW5uZXIiLCJwcm92aWRlciI6InByb3ZpZGVyW1wicmVnaXN0cnkudGVycmFmb3JtLmlvL2l0ZXJhdGl2ZS9pdGVyYXRpdmVcIl0iLCJpbnN0YW5jZXMiOlt7InByaXZhdGUiOiIiLCJzY2hlbWFfdmVyc2lvbiI6MCwiYXR0cmlidXRlcyI6eyJuYW1lIjoiMTAgdmFsdWUgd2l0aCBcInF1b3Rlc1wiIGFuZCBzcGFjZXMiLCJsYWJlbHMiOiIiLCJpZGxlX3RpbWVvdXQiOjExLCJyZXBvIjoiIiwidG9rZW4iOiIiLCJkcml2ZXIiOiIiLCJjbG91ZCI6Ii0tLSIsImN1c3RvbV9kYXRhIjoiIiwiaWQiOiIiLCJpbWFnZSI6IiIsImluc3RhbmNlX2dwdSI6IiIsImluc3RhbmNlX2hkZF9zaXplIjoxMiwiaW5zdGFuY2VfaXAiOiIiLCJpbnN0YW5jZV9sYXVuY2hfdGltZSI6IiIsImluc3RhbmNlX3R5cGUiOiIiLCJyZWdpb24iOiI5IHZhbHVlIHdpdGggXCJxdW90ZXNcIiBhbmQgc3BhY2VzIiwic3NoX25hbWUiOiIiLCJzc2hfcHJpdmF0ZSI6IiIsInNzaF9wdWJsaWMiOiIiLCJhd3Nfc2VjdXJpdHlfZ3JvdXAiOiIifX1dfQ==


EOF'
sudo chmod +x /usr/bin/cml.sh

sudo bash -c 'cat << EOF > /etc/systemd/system/cml.service
[Unit]
  Description=cml service

[Service]
  Type=oneshot
  RemainAfterExit=yes
  ExecStart=/usr/bin/cml.sh

[Install]
  WantedBy=multi-user.target
EOF'
sudo chmod +x /etc/systemd/system/cml.service

sudo systemctl daemon-reload
sudo systemctl enable cml.service --now

`
expectedProvisionerCodeCloudAWS string = `#!/bin/sh
export DEBIAN_FRONTEND=noninteractive



sudo npm install -g git+https://github.com/iterative/cml.git

sudo bash -c 'cat << EOF > /usr/bin/cml.sh
#!/bin/sh



export AWS_SECRET_ACCESS_KEY='1 value with "quotes" and spaces'
export AWS_ACCESS_KEY_ID='2 value with "quotes" and spaces'






cml-runner --name '10 value with "quotes" and spaces' --labels '16 value with "quotes" and spaces' --idle-timeout 11 --driver '15 value with "quotes" and spaces' --repo '14 value with "quotes" and spaces' --token '13 value with "quotes" and spaces' --tf_resource=eyJtb2RlIjoibWFuYWdlZCIsInR5cGUiOiJpdGVyYXRpdmVfY21sX3J1bm5lciIsIm5hbWUiOiJydW5uZXIiLCJwcm92aWRlciI6InByb3ZpZGVyW1wicmVnaXN0cnkudGVycmFmb3JtLmlvL2l0ZXJhdGl2ZS9pdGVyYXRpdmVcIl0iLCJpbnN0YW5jZXMiOlt7InByaXZhdGUiOiIiLCJzY2hlbWFfdmVyc2lvbiI6MCwiYXR0cmlidXRlcyI6eyJuYW1lIjoiMTAgdmFsdWUgd2l0aCBcInF1b3Rlc1wiIGFuZCBzcGFjZXMiLCJsYWJlbHMiOiIiLCJpZGxlX3RpbWVvdXQiOjExLCJyZXBvIjoiIiwidG9rZW4iOiIiLCJkcml2ZXIiOiIiLCJjbG91ZCI6ImF3cyIsImN1c3RvbV9kYXRhIjoiIiwiaWQiOiIiLCJpbWFnZSI6IiIsImluc3RhbmNlX2dwdSI6IiIsImluc3RhbmNlX2hkZF9zaXplIjoxMiwiaW5zdGFuY2VfaXAiOiIiLCJpbnN0YW5jZV9sYXVuY2hfdGltZSI6IiIsImluc3RhbmNlX3R5cGUiOiIiLCJyZWdpb24iOiI5IHZhbHVlIHdpdGggXCJxdW90ZXNcIiBhbmQgc3BhY2VzIiwic3NoX25hbWUiOiIiLCJzc2hfcHJpdmF0ZSI6IiIsInNzaF9wdWJsaWMiOiIiLCJhd3Nfc2VjdXJpdHlfZ3JvdXAiOiIifX1dfQ==


EOF'
sudo chmod +x /usr/bin/cml.sh

sudo bash -c 'cat << EOF > /etc/systemd/system/cml.service
[Unit]
  Description=cml service

[Service]
  Type=oneshot
  RemainAfterExit=yes
  ExecStart=/usr/bin/cml.sh

[Install]
  WantedBy=multi-user.target
EOF'
sudo chmod +x /etc/systemd/system/cml.service

sudo systemctl daemon-reload
sudo systemctl enable cml.service --now

`
expectedProvisionerCodeCloudAzure string = `#!/bin/sh
export DEBIAN_FRONTEND=noninteractive


echo "APT::Get::Assume-Yes \"true\";" | sudo tee -a /etc/apt/apt.conf.d/90assumeyes

sudo apt remove unattended-upgrades
systemctl disable apt-daily-upgrade.service

sudo add-apt-repository universe -y
sudo add-apt-repository ppa:git-core/ppa -y
sudo apt update && sudo apt-get install -y git
sudo curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
sudo usermod -aG docker ubuntu
sudo setfacl --modify user:ubuntu:rw /var/run/docker.sock

curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
sudo apt update && sudo apt-get install -y terraform

curl -sL https://deb.nodesource.com/setup_12.x | sudo bash
sudo apt update && sudo apt-get install -y nodejs

sudo apt install -y ubuntu-drivers-common git
sudo ubuntu-drivers autoinstall

curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu18.04/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt update && sudo apt install -y nvidia-docker2

sudo systemctl restart docker

sudo nvidia-smi
sudo docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi


sudo npm install -g git+https://github.com/iterative/cml.git

sudo bash -c 'cat << EOF > /usr/bin/cml.sh
#!/bin/sh





export AZURE_CLIENT_ID='3 value with "quotes" and spaces'
export AZURE_CLIENT_SECRET='4 value with "quotes" and spaces'
export AZURE_SUBSCRIPTION_ID='5 value with "quotes" and spaces'
export AZURE_TENANT_ID='6 value with "quotes" and spaces'




cml-runner --name '10 value with "quotes" and spaces' --labels '16 value with "quotes" and spaces' --idle-timeout 11 --driver '15 value with "quotes" and spaces' --repo '14 value with "quotes" and spaces' --token '13 value with "quotes" and spaces' --tf_resource=eyJtb2RlIjoibWFuYWdlZCIsInR5cGUiOiJpdGVyYXRpdmVfY21sX3J1bm5lciIsIm5hbWUiOiJydW5uZXIiLCJwcm92aWRlciI6InByb3ZpZGVyW1wicmVnaXN0cnkudGVycmFmb3JtLmlvL2l0ZXJhdGl2ZS9pdGVyYXRpdmVcIl0iLCJpbnN0YW5jZXMiOlt7InByaXZhdGUiOiIiLCJzY2hlbWFfdmVyc2lvbiI6MCwiYXR0cmlidXRlcyI6eyJuYW1lIjoiMTAgdmFsdWUgd2l0aCBcInF1b3Rlc1wiIGFuZCBzcGFjZXMiLCJsYWJlbHMiOiIiLCJpZGxlX3RpbWVvdXQiOjExLCJyZXBvIjoiIiwidG9rZW4iOiIiLCJkcml2ZXIiOiIiLCJjbG91ZCI6ImF6dXJlIiwiY3VzdG9tX2RhdGEiOiIiLCJpZCI6IiIsImltYWdlIjoiIiwiaW5zdGFuY2VfZ3B1IjoiIiwiaW5zdGFuY2VfaGRkX3NpemUiOjEyLCJpbnN0YW5jZV9pcCI6IiIsImluc3RhbmNlX2xhdW5jaF90aW1lIjoiIiwiaW5zdGFuY2VfdHlwZSI6IiIsInJlZ2lvbiI6IjkgdmFsdWUgd2l0aCBcInF1b3Rlc1wiIGFuZCBzcGFjZXMiLCJzc2hfbmFtZSI6IiIsInNzaF9wcml2YXRlIjoiIiwic3NoX3B1YmxpYyI6IiIsImF3c19zZWN1cml0eV9ncm91cCI6IiJ9fV19


EOF'
sudo chmod +x /usr/bin/cml.sh

sudo bash -c 'cat << EOF > /etc/systemd/system/cml.service
[Unit]
  Description=cml service

[Service]
  Type=oneshot
  RemainAfterExit=yes
  ExecStart=/usr/bin/cml.sh

[Install]
  WantedBy=multi-user.target
EOF'
sudo chmod +x /etc/systemd/system/cml.service

sudo systemctl daemon-reload
sudo systemctl enable cml.service --now

`
expectedProvisionerCodeCloudKubernetes string = `#!/bin/sh
export DEBIAN_FRONTEND=noninteractive









export KUBERNETES_CONFIGURATION='7 value with "quotes" and spaces'


cml-runner --name '10 value with "quotes" and spaces' --labels '16 value with "quotes" and spaces' --idle-timeout 11 --driver '15 value with "quotes" and spaces' --repo '14 value with "quotes" and spaces' --token '13 value with "quotes" and spaces' --tf_resource=eyJtb2RlIjoibWFuYWdlZCIsInR5cGUiOiJpdGVyYXRpdmVfY21sX3J1bm5lciIsIm5hbWUiOiJydW5uZXIiLCJwcm92aWRlciI6InByb3ZpZGVyW1wicmVnaXN0cnkudGVycmFmb3JtLmlvL2l0ZXJhdGl2ZS9pdGVyYXRpdmVcIl0iLCJpbnN0YW5jZXMiOlt7InByaXZhdGUiOiIiLCJzY2hlbWFfdmVyc2lvbiI6MCwiYXR0cmlidXRlcyI6eyJuYW1lIjoiMTAgdmFsdWUgd2l0aCBcInF1b3Rlc1wiIGFuZCBzcGFjZXMiLCJsYWJlbHMiOiIiLCJpZGxlX3RpbWVvdXQiOjExLCJyZXBvIjoiIiwidG9rZW4iOiIiLCJkcml2ZXIiOiIiLCJjbG91ZCI6Imt1YmVybmV0ZXMiLCJjdXN0b21fZGF0YSI6IiIsImlkIjoiIiwiaW1hZ2UiOiIiLCJpbnN0YW5jZV9ncHUiOiIiLCJpbnN0YW5jZV9oZGRfc2l6ZSI6MTIsImluc3RhbmNlX2lwIjoiIiwiaW5zdGFuY2VfbGF1bmNoX3RpbWUiOiIiLCJpbnN0YW5jZV90eXBlIjoiIiwicmVnaW9uIjoiOSB2YWx1ZSB3aXRoIFwicXVvdGVzXCIgYW5kIHNwYWNlcyIsInNzaF9uYW1lIjoiIiwic3NoX3ByaXZhdGUiOiIiLCJzc2hfcHVibGljIjoiIiwiYXdzX3NlY3VyaXR5X2dyb3VwIjoiIn19XX0=


`
)

func environmentTestData(t *testing.T) map[string]string {
	return map[string]string{
		"AWS_SECRET_ACCESS_KEY": "1 value with \"quotes\" and spaces",
	    "AWS_ACCESS_KEY_ID": "2 value with \"quotes\" and spaces",
	    "AZURE_CLIENT_ID": "3 value with \"quotes\" and spaces",
	    "AZURE_CLIENT_SECRET": "4 value with \"quotes\" and spaces",
	    "AZURE_SUBSCRIPTION_ID": "5 value with \"quotes\" and spaces",
	    "AZURE_TENANT_ID": "6 value with \"quotes\" and spaces",
	    "KUBERNETES_CONFIGURATION": "7 value with \"quotes\" and spaces",
	}
}

func schemaTestData(cloud string, t *testing.T) *schema.ResourceData {
	return schema.TestResourceDataRaw(t, resourceRunner().Schema, map[string]interface{}{
		"cloud": cloud,
		"region": "9 value with \"quotes\" and spaces",
		"name": "10 value with \"quotes\" and spaces",
		"idle_timeout": 11,
		"instance_hdd_size": 12,
		"token": "13 value with \"quotes\" and spaces",
		"repo": "14 value with \"quotes\" and spaces",
		"driver": "15 value with \"quotes\" and spaces",
		"labels": "16 value with \"quotes\" and spaces",
		"instance_gpu": "17 value with \"quotes\" and spaces",
    })
}

func testProvisionerCodeCloud(t *testing.T, cloud string, expected string) {
	for key, value := range environmentTestData(t) {
		os.Setenv(key, value)
	}
	val, err := provisionerCode(schemaTestData(cloud, t))
    if err != nil {
        t.Fail()
	} else if val != expected {
		t.Fail()
	}
}

func TestProvisionerCode_AWS(t *testing.T) {
	testProvisionerCodeCloud(t, "aws", expectedProvisionerCodeCloudAWS)
}

func TestProvisionerCode_Azure(t *testing.T) {
	testProvisionerCodeCloud(t, "azure", expectedProvisionerCodeCloudAzure)
}

func TestProvisionerCode_Kubernetes(t *testing.T) {
	testProvisionerCodeCloud(t, "kubernetes", expectedProvisionerCodeCloudKubernetes)
}

func TestProvisionerCode_Invalid(t *testing.T) {
	testProvisionerCodeCloud(t, "---", expectedProvisionerCodeCloudInvalid)
}
