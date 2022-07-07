#!/bin/bash

# check for jq
if ! command -v jq &> /dev/null ; then
    echo "jq is missing, installing."
    apt-get update 1>/dev/null && sudo apt-get install -y jq 1>/dev/null
fi
# check for git
if ! command -v git &>/dev/null ; then
    echo "git is missing, installing."
    apt-get update 1>/dev/null && sudo apt-get install -y git 1>/dev/null
fi
# check for GH cli
if ! command -v gh &>/dev/null ; then
    echo "Missing gh cli, installing."
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
    apt-get update 1>/dev/null && sudo apt-get install -y gh 1>/dev/null
fi
# check for docker
if ! command -v docker &> /dev/null ; then
    curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh
    usermod -aG docker ubuntu
fi
# check for PAT
if [ -z "$GITHUB_PAT" ]; then
    echo "Please set locally set env GITHUB_PAT."
    echo 'ex: export GITHUB_PAT="ghp_***"'
    exit 1
fi

# GitHub user json
gh_user_json=$(
    curl --silent \
        --header "Accept: application/vnd.github.v3+json" \
        --header "Authorization: Bearer $GITHUB_PAT" \
        https://api.github.com/user
)
# GitHub info
gh_username=$(echo "$gh_user_json" | jq .login | tr -d \")
gh_name=$(echo "$gh_user_json" | jq .name | tr -d \")
gh_email=$(echo "$gh_user_json" | jq .email | tr -d \")

# Create new user
#useradd -m "$gh_username"

# Add users ssh keys
curl "https://github.com/$gh_username.keys" >> "/home/ubuntu/.ssh/authorized_keys"

# basic reqs
apt-get install -y build-essential python3-pip virtualenv pipenv nvtop > main_apt_install.log

# change user
su - ubuntu <<EOS
cd ~
# basic git setup from GitHub user/email
git config --global user.name "$gh_name"
git config --global user.email "$gh_email"

# setup gh auth helper for commiting
echo "$GITHUB_PAT" | gh auth login --hostname github.com --with-token
gh auth setup-git

# Clone repo
git clone https://github.com/GIT_ORG/GIT_REPO.git
cd GIT_REPO

if [ ! -e ".devcontainer.json" ]; then
    echo "found dev container, use that."

    # detect python setup
    if [ -e "requirements.txt" ]; then
        virtualenv .venv
        source .venv/bin/activate
        pip install -r requirements.txt
    fi
    if [ -e "Pipfile" ]; then
        pipenv install --skip-lock
    fi
fi
EOS

#apt-get update >> ./apt_update.log && apt-get install -y git build-essential nvtop >> ./apt_install.log
#systemd-run --no-block --service-type=exec bash -c 'curl https://gist.githubusercontent.com/dacbd/c527d1a214f7118e6d66e52a6abb4c4f/raw/db3cba14dcc4a23fb1b7c7a115563942d4164aaf/nvidia-src-setup.sh | bash'
#apt-get install -y python-is-python3 python3-pip pipenv >> ./apt_install.log
#pushd /home/ubuntu
#sudo --user=ubuntu git clone https://github.com/iterative/magnetic-tiles-defect.git
#pushd magnetic-tiles-defect
#sudo --user=ubuntu pipenv install --skip-lock
echo "***READY***"
sleep infinity
