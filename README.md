# res
Tool for filtering kubernetes resources defined in yaml


# install
On mac
```
brew install profiprog/tap/res
```
On linux
```
VERSION=v0.1.0
curl -sL https://github.com/profiprog/res/releases/download/$VERSION/res_$VERSION_linux_amd64.tar.gz | sudo tar xz -C /usr/local/bin res
```
Docker image
```
docker pull ghcr.io/profiprog/res
```

# Usage
```
res --help
curl -sL https://raw.githubusercontent.com/argoproj/argo-cd/master/manifests/ha/install.yaml | res -
```

# Use as container
```
docker run --rm -t ghcr.io/profiprog/res -h
docker run --rm -t -v "$PWD:/workdir" ghcr.io/profiprog/res -i=/workdir -
curl -sL https://raw.githubusercontent.com/argoproj/argo-cd/master/manifests/ha/install.yaml | docker run --rm -i -v "$PWD:/workdir" ghcr.io/profiprog/res - Role
```